package tools

import (
	"context"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/getditto/ditto-go-sdk/ditto"
)

// PresenceArgs holds arguments for the presence command
type PresenceArgs struct {
	PeerScope  string `json:"peer_scope"`
	TimeoutSec int32  `json:"timeout_sec"`
}

// DoPresence observes local and/or remote peer metadata
func DoPresence(ctx context.Context, config *Config, args *PresenceArgs) error {
	ctx, task := trace.NewTask(ctx, "DoPresence")
	defer task.End()
	defer trace.StartRegion(ctx, "DoPresence").End()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	toolkit, err := NewToolkit(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create toolkit: %w", err)
	}
	defer toolkit.Close()

	if err := toolkit.StartSync(); err != nil {
		return fmt.Errorf("failed to start sync: %w", err)
	}

	// Channel to receive presence updates
	type PresenceUpdate struct {
		graph *ditto.PresenceGraph
	}
	presenceChan := make(chan PresenceUpdate, 20)

	// Start observing presence
	observer, err := toolkit.RegisterPresenceObserver(
		func(graph *ditto.PresenceGraph) {
			select {
			case presenceChan <- PresenceUpdate{graph: graph}:
			default:
				// Channel is full, skip this update to prevent blocking
			}
		},
	)
	if err != nil {
		return fmt.Errorf("failed to observe presence: %w", err)
	}
	defer observer.Stop()

	fmt.Printf("Observing %s peers (press Ctrl+C to stop", args.PeerScope)
	if args.TimeoutSec > 0 {
		fmt.Printf(" or wait %d seconds", args.TimeoutSec)
	}
	fmt.Println(")...")

	// Quit on timeout or SIGINT/SIGTERM
	quit := timeoutOrInterrupt(time.Duration(args.TimeoutSec) * time.Second)

	// Main event loop
	for {
		select {
		case update := <-presenceChan:
			displayPresenceGraph(update.graph, args.PeerScope)
		case <-quit:
			fmt.Println("\nStopping presence observation...")
			return nil
		}
	}
}

// displayPresenceGraph displays the presence graph based on the peer scope
func displayPresenceGraph(graph *ditto.PresenceGraph, peerScope string) {
	fmt.Println("\n---- PRESENCE UPDATE ----")
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	fmt.Printf("Timestamp: %s\n", timestamp)

	// Display based on peer scope
	switch peerScope {
	case PeerScopeLocal:
		// Only show local peer
		fmt.Println("\nLocal Peer:")
		displayPeerInfo(graph.LocalPeer)

	case PeerScopeRemote:
		// Only show remote peers
		fmt.Printf("\nRemote Peers (%d):\n", len(graph.RemotePeers))
		if len(graph.RemotePeers) > 0 {
			for _, peer := range graph.RemotePeers {
				displayPeerInfo(peer)
			}
		} else {
			fmt.Println("  <none>")
		}

	case PeerScopeAll:
		// Show both local and remote peers
		fmt.Println("\nLocal Peer:")
		displayPeerInfo(graph.LocalPeer)

		fmt.Printf("\nRemote Peers (%d):\n", len(graph.RemotePeers))
		if len(graph.RemotePeers) > 0 {
			for _, peer := range graph.RemotePeers {
				displayPeerInfo(peer)
			}
		} else {
			fmt.Println("  <none>")
		}

		// Also show connections in "all" mode
		fmt.Printf("\nConnections (%d):\n", len(graph.AllConnectionsByID))
		if len(graph.AllConnectionsByID) > 0 {
			for _, conn := range graph.AllConnectionsByID {
				displayConnectionInfo(conn)
			}
		} else {
			fmt.Println("  <none>")
		}

	default:
		fmt.Printf("Unknown peer scope: %s\n", peerScope)
	}

	fmt.Println("---- END PRESENCE UPDATE ----")
}

// displayPeerInfo displays information about a single peer
func displayPeerInfo(peer *ditto.Peer) {
	if peer == nil {
		fmt.Println("  <nil peer>")
		return
	}

	fmt.Printf("  PeerKey: %s\n", peer.PeerKeyString)
	fmt.Printf("    Device: %s\n", peer.DeviceName)
	fmt.Printf("    OS: %s\n", peer.OS)
	fmt.Printf("    SDK: %s\n", peer.DittoSDKVersion)
	fmt.Printf("    Connected: %v\n", peer.IsConnectedToDittoCloud)

	// Display connections
	if len(peer.Connections) > 0 {
		fmt.Printf("    Connections: %d\n", len(peer.Connections))
		for _, conn := range peer.Connections {
			if conn != nil {
				fmt.Printf("      - %s\n", conn.ID)
			}
		}
	}

	// Display peer metadata if available
	if peer.PeerMetadata != nil && len(peer.PeerMetadata) > 0 {
		fmt.Printf("    Metadata: %v\n", peer.PeerMetadata)
	}
}

// displayConnectionInfo displays information about a connection between peers
func displayConnectionInfo(conn *ditto.Connection) {
	if conn == nil {
		return
	}
	fmt.Printf(
		"  %s <-> %s [%s]\n",
		conn.PeerKeyString1,
		conn.PeerKeyString2,
		conn.ID,
	)
}
