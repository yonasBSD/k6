package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"go.k6.io/k6/cmd/state"
	"go.k6.io/k6/errext"
	"go.k6.io/k6/errext/exitcodes"
	"go.k6.io/k6/ext"
	"go.k6.io/k6/internal/log"
	"go.k6.io/k6/secretsource"

	_ "go.k6.io/k6/internal/secretsource" // import it to register internal secret sources
)

const waitLoggerCloseTimeout = time.Second * 5

// ExecuteWithGlobalState runs the root command with an existing GlobalState.
// It adds all child commands to the root command and it sets flags appropriately.
// It is called by main.main(). It only needs to happen once to the rootCmd.
func ExecuteWithGlobalState(gs *state.GlobalState) {
	newRootCommand(gs).execute()
}

// This is to keep all fields needed for the main/root k6 command
type rootCommand struct {
	globalState *state.GlobalState

	cmd            *cobra.Command
	stopLoggersCh  chan struct{}
	loggersWg      sync.WaitGroup
	loggerIsRemote bool
	launcher       *launcher
}

// newRootCommand creates a root command with a default launcher
func newRootCommand(gs *state.GlobalState) *rootCommand {
	return newRootWithLauncher(gs, newLauncher(gs))
}

// newRootWithLauncher creates a root command with a launcher.
// It facilitates unit testing scenarios.
func newRootWithLauncher(gs *state.GlobalState, l *launcher) *rootCommand {
	c := &rootCommand{
		globalState:   gs,
		stopLoggersCh: make(chan struct{}),
		launcher:      l,
	}
	// the base command when called without any subcommands.
	rootCmd := &cobra.Command{
		Use:               gs.BinaryName,
		Short:             "a next-generation load generator",
		Long:              "\n" + getBanner(gs.Flags.NoColor || !gs.Stdout.IsTTY, isTrueColor(gs.Env)),
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: c.persistentPreRunE,
		Version:           versionString(),
	}

	rootCmd.SetVersionTemplate(
		`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "v%s\n" .Version}}`,
	)

	usageTemplate := rootCmd.UsageTemplate()
	usageTemplate = strings.ReplaceAll(usageTemplate, "FlagUsages", "FlagUsagesWrapped 120")
	rootCmd.SetUsageTemplate(usageTemplate)

	rootCmd.PersistentFlags().AddFlagSet(rootCmdPersistentFlagSet(gs))
	rootCmd.SetArgs(gs.CmdArgs[1:])
	rootCmd.SetOut(gs.Stdout)
	rootCmd.SetErr(gs.Stderr) // TODO: use gs.logger.WriterLevel(logrus.ErrorLevel)?
	rootCmd.SetIn(gs.Stdin)

	subCommands := []func(*state.GlobalState) *cobra.Command{
		getCmdArchive, getCmdCloud, getCmdNewScript, getCmdInspect,
		getCmdLogin, getCmdPause, getCmdResume, getCmdScale, getCmdRun,
		getCmdStats, getCmdStatus, getCmdVersion,
	}

	for _, sc := range subCommands {
		rootCmd.AddCommand(sc(gs))
	}

	c.cmd = rootCmd
	return c
}

func (c *rootCommand) persistentPreRunE(cmd *cobra.Command, args []string) error {
	err := c.setupLoggers(c.stopLoggersCh)
	if err != nil {
		return err
	}

	c.globalState.Logger.Debugf("k6 version: v%s", fullVersion())

	// If binary provisioning is not enabled, continue with the regular k6 execution path
	if !c.globalState.Flags.BinaryProvisioning {
		c.globalState.Logger.Debug("Binary Provisioning feature is disabled.")
		return nil
	}

	c.globalState.Logger.
		Debug("Binary Provisioning feature is enabled.")

	return c.launcher.launch(cmd, args)
}

func (c *rootCommand) execute() {
	ctx, cancel := context.WithCancel(c.globalState.Ctx)
	c.globalState.Ctx = ctx

	exitCode := -1
	defer func() {
		cancel()
		c.stopLoggers()
		c.globalState.OSExit(exitCode)
	}()

	defer func() {
		if r := recover(); r != nil {
			exitCode = int(exitcodes.GoPanic)
			err := fmt.Errorf("unexpected k6 panic: %s\n%s", r, debug.Stack())
			if c.loggerIsRemote {
				c.globalState.FallbackLogger.Error(err)
			}
			c.globalState.Logger.Error(err)
		}
	}()

	err := c.cmd.Execute()
	if err == nil {
		exitCode = 0
		return
	}

	var ecerr errext.HasExitCode
	if errors.As(err, &ecerr) {
		exitCode = int(ecerr.ExitCode())
	}

	errText, fields := errext.Format(err)
	c.globalState.Logger.WithFields(fields).Error(errText)
	if c.loggerIsRemote {
		c.globalState.FallbackLogger.WithFields(fields).Error(errText)
	}
}

func (c *rootCommand) stopLoggers() {
	done := make(chan struct{})
	go func() {
		c.loggersWg.Wait()
		close(done)
	}()
	close(c.stopLoggersCh)
	select {
	case <-done:
	case <-time.After(waitLoggerCloseTimeout):
		c.globalState.FallbackLogger.Errorf("The logger didn't stop in %s", waitLoggerCloseTimeout)
	}
}

func rootCmdPersistentFlagSet(gs *state.GlobalState) *pflag.FlagSet {
	flags := pflag.NewFlagSet("", pflag.ContinueOnError)
	// TODO: refactor this config, the default value management with pflag is
	// simply terrible... :/
	//
	// We need to use `gs.Flags.<value>` both as the destination and as
	// the value here, since the config values could have already been set by
	// their respective environment variables. However, we then also have to
	// explicitly set the DefValue to the respective default value from
	// `gs.DefaultFlags.<value>`, so that the `k6 --help` message is
	// not messed up...

	// TODO(@mstoykov): likely needs work - no env variables and such. No config.json.
	flags.StringArrayVar(&gs.Flags.SecretSource, "secret-source", gs.Flags.SecretSource,
		"setting secret sources for k6 file[=./path.fileformat],")

	flags.StringVar(&gs.Flags.LogOutput, "log-output", gs.Flags.LogOutput,
		"change the output for k6 logs, possible values are: "+
			"'stderr', 'stdout', 'none', 'loki[=host:port]', 'file[=./path.fileformat]'")
	flags.Lookup("log-output").DefValue = gs.DefaultFlags.LogOutput

	flags.StringVar(&gs.Flags.LogFormat, "log-format", gs.Flags.LogFormat, "log output format")
	flags.Lookup("log-format").DefValue = gs.DefaultFlags.LogFormat

	flags.StringVarP(&gs.Flags.ConfigFilePath, "config", "c", gs.Flags.ConfigFilePath, "JSON config file")
	// And we also need to explicitly set the default value for the usage message here, so things
	// like `K6_CONFIG="blah" k6 run -h` don't produce a weird usage message
	flags.Lookup("config").DefValue = gs.DefaultFlags.ConfigFilePath
	must(cobra.MarkFlagFilename(flags, "config"))

	flags.BoolVar(&gs.Flags.NoColor, "no-color", gs.Flags.NoColor, "disable colored output")
	flags.Lookup("no-color").DefValue = strconv.FormatBool(gs.DefaultFlags.NoColor)

	// TODO: support configuring these through environment variables as well?
	// either with croconf or through the hack above...
	flags.BoolVarP(&gs.Flags.Verbose, "verbose", "v", gs.DefaultFlags.Verbose, "enable verbose logging")
	flags.BoolVarP(&gs.Flags.Quiet, "quiet", "q", gs.DefaultFlags.Quiet, "disable progress updates")
	flags.StringVarP(&gs.Flags.Address, "address", "a", gs.DefaultFlags.Address, "address for the REST API server")
	flags.BoolVar(
		&gs.Flags.ProfilingEnabled,
		"profiling-enabled",
		gs.DefaultFlags.ProfilingEnabled,
		"enable profiling (pprof) endpoints, k6's REST API should be enabled as well",
	)

	return flags
}

// RawFormatter it does nothing with the message just prints it
type RawFormatter struct{}

// Format renders a single log entry
func (f RawFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return append([]byte(entry.Message), '\n'), nil
}

// The returned channel will be closed when the logger has finished flushing and pushing logs after
// the provided context is closed. It is closed if the logger isn't buffering and sending messages
// Asynchronously
func (c *rootCommand) setupLoggers(stop <-chan struct{}) error {
	if c.globalState.Flags.Verbose {
		c.globalState.Logger.SetLevel(logrus.DebugLevel)
	}

	var (
		hook log.AsyncHook
		err  error
	)

	loggerForceColors := false // disable color by default
	switch line := c.globalState.Flags.LogOutput; {
	case line == "stderr":
		loggerForceColors = !c.globalState.Flags.NoColor && c.globalState.Stderr.IsTTY
		c.globalState.Logger.SetOutput(c.globalState.Stderr)
	case line == "stdout":
		loggerForceColors = !c.globalState.Flags.NoColor && c.globalState.Stdout.IsTTY
		c.globalState.Logger.SetOutput(c.globalState.Stdout)
	case line == "none":
		c.globalState.Logger.SetOutput(io.Discard)
	case strings.HasPrefix(line, "loki"):
		c.loggerIsRemote = true
		hook, err = log.LokiFromConfigLine(c.globalState.FallbackLogger, line)
		if err != nil {
			return err
		}
		c.globalState.Flags.LogFormat = "raw"
	case strings.HasPrefix(line, "file"):
		hook, err = log.FileHookFromConfigLine(
			c.globalState.FS, c.globalState.Getwd,
			c.globalState.FallbackLogger, line,
		)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported log output '%s'", line)
	}

	switch c.globalState.Flags.LogFormat {
	case "raw":
		c.globalState.Logger.SetFormatter(&RawFormatter{})
		c.globalState.Logger.Debug("Logger format: RAW")
	case "json":
		c.globalState.Logger.SetFormatter(&logrus.JSONFormatter{})
		c.globalState.Logger.Debug("Logger format: JSON")
	default:
		c.globalState.Logger.SetFormatter(&logrus.TextFormatter{
			ForceColors: loggerForceColors, DisableColors: c.globalState.Flags.NoColor,
		})
		c.globalState.Logger.Debug("Logger format: TEXT")
	}

	secretsources, err := createSecretSources(c.globalState)
	if err != nil {
		return err
	}
	// it is important that we add this hook first as hooks are executed in order of addition
	// and this means no other hook will get secrets
	var secretsHook logrus.Hook
	c.globalState.SecretsManager, secretsHook, err = secretsource.NewManager(secretsources)
	if err != nil {
		return err
	}
	if len(secretsources) != 0 {
		// don't actually filter anything if there will be no secrets
		c.globalState.Logger.AddHook(secretsHook)
	}

	cancel := func() {} // noop as default
	if hook != nil {
		ctx := context.Background()
		ctx, cancel = context.WithCancel(ctx)
		c.setLoggerHook(ctx, hook)
	}

	// Sometimes the Go runtime uses the standard log output to
	// log some messages directly.
	// It does when an invalid char is found in a Cookie.
	// Check for details https://github.com/grafana/k6/issues/711#issue-341414887
	w := c.globalState.Logger.Writer()
	stdlog.SetOutput(w)
	c.loggersWg.Add(1)
	go func() {
		<-stop
		cancel()
		_ = w.Close()
		c.loggersWg.Done()
	}()
	return nil
}

func (c *rootCommand) setLoggerHook(ctx context.Context, h log.AsyncHook) {
	c.loggersWg.Add(1)
	go func() {
		h.Listen(ctx)
		c.loggersWg.Done()
	}()
	c.globalState.Logger.AddHook(h)
	c.globalState.Logger.SetOutput(io.Discard) // don't output to anywhere else
}

func createSecretSources(gs *state.GlobalState) (map[string]secretsource.Source, error) {
	baseParams := secretsource.Params{
		Logger:      gs.Logger,
		Environment: gs.Env,
		FS:          gs.FS,
		Usage:       gs.Usage,
	}

	result := make(map[string]secretsource.Source)
	for _, line := range gs.Flags.SecretSource {
		t, config, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("couldn't parse secret source configuration %q", line)
		}
		secretSources := ext.Get(ext.SecretSourceExtension)
		found, ok := secretSources[t]
		if !ok {
			return nil, fmt.Errorf("no secret source for type %q for configuration %q", t, line)
		}
		c := found.Module.(secretsource.Constructor) //nolint:forcetypeassert
		params := baseParams
		name, isDefault, config := extractNameAndDefault(config)
		params.ConfigArgument = config

		secretSource, err := c(params)
		if err != nil {
			return nil, err
		}
		_, alreadRegistered := result[name]
		if alreadRegistered {
			return nil, fmt.Errorf("secret source for name %q already registered before configuration %q", t, line)
		}
		result[name] = secretSource
		if isDefault {
			if _, ok := result["default"]; ok {
				return nil, fmt.Errorf("can't have two secret sources that are default ones, second one was %q", config)
			}
			result["default"] = secretSource
		}
	}

	if len(result) == 1 {
		for _, l := range result {
			result["default"] = l
		}
	}

	return result, nil
}

func extractNameAndDefault(config string) (name string, isDefault bool, remaining string) {
	list := strings.Split(config, ",")
	remainingArray := make([]string, 0, len(list))
	for _, kv := range list {
		if kv == "default" {
			isDefault = true
			continue
		}
		k, v, _ := strings.Cut(kv, "=")
		if k == "name" {
			name = v
			continue
		}
		remainingArray = append(remainingArray, kv)
	}
	return name, isDefault, strings.Join(remainingArray, ",")
}
