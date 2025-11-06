// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ditto

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// Presence is the entrypoint for all actions that relate presence of other peers known by the current peer, either
// directly or through other peers.
//
// Presence can be accessed from a Ditto instance via its Ditto.Presence() method.
type Presence struct {
	dittoHandle *ffi.DittoHandle

	mu        sync.RWMutex
	observers map[string]*PresenceObserver
}

// newPresence creates a new presence manager.
//
// This is called internally by Ditto during initialization.
// Applications should not call this directly.
func newPresence(dittoHandle *ffi.DittoHandle) *Presence {
	return &Presence{
		dittoHandle: dittoHandle,
		observers:   make(map[string]*PresenceObserver),
	}
}

// Graph returns the current presence graph capturing all known peers and connections between them.
func (p *Presence) Graph() *PresenceGraph {
	sdkDebugTrace("gosdk: Presence.Graph")

	graphJSON, err := ffi.GetPresenceGraph(p.dittoHandle)
	if err != nil {
		return nil
	}

	graph, err := p.parsePresenceGraph(graphJSON)
	if err != nil {
		LogErrorF("gosdk: Presence.Graph received invalid data: %v", err)
		sdkDebugTraceF("gosdk: raw presence data: %s", string(graphJSON))
	}
	return graph
}

// Observe requests information about Ditto peers in range of this device.
//
// This method returns an observer which should be held as long as updates are required.
// Call the PresenceObserver.Stop() method to cease updates.
//
// A newly registered observer will have a peers update delivered to it immediately. From then on it will be invoked
// repeatedly when Ditto devices come and go, or the active connections to them change.
func (p *Presence) Observe(callback PresenceObservationHandler) *PresenceObserver {
	sdkDebugTrace("gosdk: Presence.Observe called")

	p.mu.Lock()
	defer p.mu.Unlock()

	// Create observer even if FFI fails (for testing and fallback)
	observer := newPresenceObserver(callback)

	ffiCallback := func(data []byte) {
		sdkDebugTrace("gosdk: Presence.Observe callback called")

		newGraph, err := p.parsePresenceGraph(data)
		if err != nil {
			LogErrorF("gosdk: Presence.Observe callback received invalid data: %s", err)
			sdkDebugTraceF("gosdk: raw presence data: %s", string(data))
			return
		}

		sdkDebugTrace("gosdk: calling user presence handler")
		observer.handler(newGraph)
		sdkDebugTrace("gosdk: user presence handler returned")
	}

	observerHandle, err := ffi.RegisterPresenceObserver(p.dittoHandle, ffiCallback)
	if err != nil {
		LogErrorF("gosdk: failed to register presence observer: %v", err)
	}
	// Continue even if FFI registration fails
	observer.handle = observerHandle

	p.observers[observer.id] = observer
	return observer
}

func (p *Presence) parsePresenceGraph(graphJSON []byte) (*PresenceGraph, error) {
	var graph PresenceGraph
	if err := json.Unmarshal(graphJSON, &graph); err != nil {
		return nil, err
	}

	graph.AllConnectionsByID = buildConnectionsMapFromPeers(&graph)

	return &graph, nil
}

func buildConnectionsMapFromPeers(graph *PresenceGraph) map[string]*Connection {
	// Build connections map from all peers
	allConnectionsByID := make(map[string]*Connection)
	allPeers := make([]*Peer, 0, len(graph.RemotePeers)+1)
	if graph.LocalPeer != nil {
		allPeers = append(allPeers, graph.LocalPeer)
	}
	allPeers = append(allPeers, graph.RemotePeers...)

	for _, peer := range allPeers {
		for _, conn := range peer.Connections {
			if conn != nil && conn.ID != "" {
				allConnectionsByID[conn.ID] = conn
			}
		}
	}
	return allConnectionsByID
}

// removeObserver stops a presence observer and removes it from receiving updates.
func (p *Presence) removeObserver(observer *PresenceObserver) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if observer != nil {
		observer.Stop()
		delete(p.observers, observer.id)

		// Cancel FFI observer if it has a handle
		if observer.handle != nil {
			ffi.CancelPresenceObserver(observer.handle)
		}
	}
}

// SetPeerMetadata sets metadata associated with the current peer.
//
// Other peers in the same mesh can access this user-provided dictionary of metadata via the presence graph from
// Presence.Graph() and when evaluating connection requests using Presence.SetConnectionRequestHandler.
// Use SetPeerMetadata() or SetPeerMetadataJSONData() to set this value.
//
// This is a convenience property that wraps SetPeerMetadataJSONData().
func (p *Presence) SetPeerMetadata(metadata map[string]any) error {
	// Convert metadata to JSON
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return p.SetPeerMetadataJSONData(jsonData)
}

// SetPeerMetadataJSONData sets metadata from JSON data
//
// Other peers in the same mesh can access this user-provided dictionary of metadata via the presence graph from
// Presence.Graph() and when evaluating connection requests using Presence.SetConnectionRequestHandler.
// Use SetPeerMetadata() or SetPeerMetadataJSONData() to set this value.
//
// Uses UTF-8 encoding.
func (p *Presence) SetPeerMetadataJSONData(jsonData []byte) error {
	return ffi.SetPeerMetadataJSON(p.dittoHandle, jsonData)
}

// GetPeerMetadata gets metadata associated with the current peer.
//
// Other peers in the same mesh can access this user-provided dictionary of metadata via the presence graph from
// Presence.Graph() and when evaluating connection requests using Presence.SetConnectionRequestHandler.
// Use SetPeerMetadata() or SetPeerMetadataJSONData() to set this value.
//
// This is a convenience property that wraps GetPeerMetadataJSONData.
func (p *Presence) GetPeerMetadata() (map[string]any, error) {
	// Get all peer metadata as JSON
	jsonData, err := p.GetPeerMetadataJSONData()
	if err != nil {
		return nil, err
	}

	var metadata map[string]any
	if err := json.Unmarshal(jsonData, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return metadata, nil
}

// GetPeerMetadataJSONData gets metadata associated with the current peer.
//
// Other peers in the same mesh can access this user-provided dictionary of metadata via the presence graph from
// Presence.Graph() and when evaluating connection requests using Presence.SetConnectionRequestHandler.
// Use SetPeerMetadata() or SetPeerMetadataJSONData() to set this value.
//
// This is a convenience property that wraps GetPeerMetadataJSONData.
func (p *Presence) GetPeerMetadataJSONData() ([]byte, error) {
	// Get all peer metadata as JSON
	jsonData, err := ffi.GetPeerMetadataJSON(p.dittoHandle)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

// ConnectionRequestHandler is a function that handles connection requests.
type ConnectionRequestHandler func(request *ConnectionRequest) ConnectionRequestAuthorization

// SetConnectionRequestHandler registers a handler function to control which peers in a Ditto mesh can connect to the
// current peer.
//
// Each peer in a Ditto mesh will attempt to connect to other peers that it can reach. By default, the mesh will try
// and establish connections that optimize for the best overall connectivity between peers. However, you can set this
// handler to assert some control over which peers you connect to.
//
// If set, this handler is called for every incoming connection request from a remote peer and is passed the other
// peer’s PeerKeyString, PeerMetadata, and IdentityServiceMetadata. The handler can then accept or reject the request
// by returning an according DittoConnectionRequestAuthorization value. When the connection request is rejected, the
// remote peer may retry the connection request after a short delay.
//
// Connection request handlers must reliably respond to requests within a short time. If a handler takes too long to
// respond or throws an exception, the connection request will be denied. The response currently times out after 10
// seconds, but this exact value may be subject to change in future releases.
//
// Note: The handler is called from a different thread.
func (p *Presence) SetConnectionRequestHandler(handler ConnectionRequestHandler) {
	adapter := &connectionRequestHandlerAdapter{
		handler: handler,
	}
	handlerID := ffi.RegisterConnectionRequestHandler(adapter)
	ffi.SetConnectionRequestHandler(p.dittoHandle, handlerID)
}

// connectionRequestHandlerAdapter adapts between FFI and public API
type connectionRequestHandlerAdapter struct {
	handler ConnectionRequestHandler
}

// HandleConnectionRequest implements ffi.ConnectionRequestHandler
func (a *connectionRequestHandlerAdapter) HandleConnectionRequest(request *ffi.ConnectionRequestWrapper) ffi.ConnectionRequestAuthorization {
	// Convert FFI request to public API request
	apiRequest := &ConnectionRequest{
		PeerKey:      request.GetPeerKey(),
		PeerMetadata: request.GetPeerMetadata(),
		IdentityData: request.GetIdentityServiceMetadata(),
	}

	// Call the user's handler
	auth := ConnectionRequestAuthorizationAllow
	if a.handler != nil {
		auth = a.handler(apiRequest)
	}

	// Convert authorization to FFI type
	if auth == ConnectionRequestAuthorizationAllow {
		return ffi.ConnectionRequestAuthorizationAllow
	}
	return ffi.ConnectionRequestAuthorizationDeny
}

// ConnectionRequest represents a connection request from a peer
type ConnectionRequest struct {
	PeerKey      string
	PeerMetadata map[string]any
	IdentityData map[string]any
}

// ConnectionRequestAuthorization indicates whether a connection request should be authorized.
type ConnectionRequestAuthorization ffi.ConnectionRequestAuthorization

const (
	ConnectionRequestAuthorizationAllow = ConnectionRequestAuthorization(ffi.ConnectionRequestAuthorizationAllow)
	ConnectionRequestAuthorizationDeny  = ConnectionRequestAuthorization(ffi.ConnectionRequestAuthorizationDeny)
)
