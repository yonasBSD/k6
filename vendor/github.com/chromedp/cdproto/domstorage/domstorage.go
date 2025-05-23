// Package domstorage provides the Chrome DevTools Protocol
// commands, types, and events for the DOMStorage domain.
//
// Query and modify DOM storage.
//
// Generated by the cdproto-gen command.
package domstorage

// Code generated by cdproto-gen. DO NOT EDIT.

import (
	"context"

	"github.com/chromedp/cdproto/cdp"
)

// ClearParams [no description].
type ClearParams struct {
	StorageID *StorageID `json:"storageId"`
}

// Clear [no description].
//
// See: https://chromedevtools.github.io/devtools-protocol/tot/DOMStorage#method-clear
//
// parameters:
//
//	storageID
func Clear(storageID *StorageID) *ClearParams {
	return &ClearParams{
		StorageID: storageID,
	}
}

// Do executes DOMStorage.clear against the provided context.
func (p *ClearParams) Do(ctx context.Context) (err error) {
	return cdp.Execute(ctx, CommandClear, p, nil)
}

// DisableParams disables storage tracking, prevents storage events from
// being sent to the client.
type DisableParams struct{}

// Disable disables storage tracking, prevents storage events from being sent
// to the client.
//
// See: https://chromedevtools.github.io/devtools-protocol/tot/DOMStorage#method-disable
func Disable() *DisableParams {
	return &DisableParams{}
}

// Do executes DOMStorage.disable against the provided context.
func (p *DisableParams) Do(ctx context.Context) (err error) {
	return cdp.Execute(ctx, CommandDisable, nil, nil)
}

// EnableParams enables storage tracking, storage events will now be
// delivered to the client.
type EnableParams struct{}

// Enable enables storage tracking, storage events will now be delivered to
// the client.
//
// See: https://chromedevtools.github.io/devtools-protocol/tot/DOMStorage#method-enable
func Enable() *EnableParams {
	return &EnableParams{}
}

// Do executes DOMStorage.enable against the provided context.
func (p *EnableParams) Do(ctx context.Context) (err error) {
	return cdp.Execute(ctx, CommandEnable, nil, nil)
}

// GetDOMStorageItemsParams [no description].
type GetDOMStorageItemsParams struct {
	StorageID *StorageID `json:"storageId"`
}

// GetDOMStorageItems [no description].
//
// See: https://chromedevtools.github.io/devtools-protocol/tot/DOMStorage#method-getDOMStorageItems
//
// parameters:
//
//	storageID
func GetDOMStorageItems(storageID *StorageID) *GetDOMStorageItemsParams {
	return &GetDOMStorageItemsParams{
		StorageID: storageID,
	}
}

// GetDOMStorageItemsReturns return values.
type GetDOMStorageItemsReturns struct {
	Entries []Item `json:"entries,omitempty,omitzero"`
}

// Do executes DOMStorage.getDOMStorageItems against the provided context.
//
// returns:
//
//	entries
func (p *GetDOMStorageItemsParams) Do(ctx context.Context) (entries []Item, err error) {
	// execute
	var res GetDOMStorageItemsReturns
	err = cdp.Execute(ctx, CommandGetDOMStorageItems, p, &res)
	if err != nil {
		return nil, err
	}

	return res.Entries, nil
}

// RemoveDOMStorageItemParams [no description].
type RemoveDOMStorageItemParams struct {
	StorageID *StorageID `json:"storageId"`
	Key       string     `json:"key"`
}

// RemoveDOMStorageItem [no description].
//
// See: https://chromedevtools.github.io/devtools-protocol/tot/DOMStorage#method-removeDOMStorageItem
//
// parameters:
//
//	storageID
//	key
func RemoveDOMStorageItem(storageID *StorageID, key string) *RemoveDOMStorageItemParams {
	return &RemoveDOMStorageItemParams{
		StorageID: storageID,
		Key:       key,
	}
}

// Do executes DOMStorage.removeDOMStorageItem against the provided context.
func (p *RemoveDOMStorageItemParams) Do(ctx context.Context) (err error) {
	return cdp.Execute(ctx, CommandRemoveDOMStorageItem, p, nil)
}

// SetDOMStorageItemParams [no description].
type SetDOMStorageItemParams struct {
	StorageID *StorageID `json:"storageId"`
	Key       string     `json:"key"`
	Value     string     `json:"value"`
}

// SetDOMStorageItem [no description].
//
// See: https://chromedevtools.github.io/devtools-protocol/tot/DOMStorage#method-setDOMStorageItem
//
// parameters:
//
//	storageID
//	key
//	value
func SetDOMStorageItem(storageID *StorageID, key string, value string) *SetDOMStorageItemParams {
	return &SetDOMStorageItemParams{
		StorageID: storageID,
		Key:       key,
		Value:     value,
	}
}

// Do executes DOMStorage.setDOMStorageItem against the provided context.
func (p *SetDOMStorageItemParams) Do(ctx context.Context) (err error) {
	return cdp.Execute(ctx, CommandSetDOMStorageItem, p, nil)
}

// Command names.
const (
	CommandClear                = "DOMStorage.clear"
	CommandDisable              = "DOMStorage.disable"
	CommandEnable               = "DOMStorage.enable"
	CommandGetDOMStorageItems   = "DOMStorage.getDOMStorageItems"
	CommandRemoveDOMStorageItem = "DOMStorage.removeDOMStorageItem"
	CommandSetDOMStorageItem    = "DOMStorage.setDOMStorageItem"
)
