package handlers

import "testing"

func TestDefaultChannelMetadataEnablesAutoConnectAndHistorySync(t *testing.T) {
	metadata := defaultChannelMetadata(nil)

	if metadata["autoConnect"] != true {
		t.Fatalf("expected autoConnect default true, got %#v", metadata["autoConnect"])
	}
	if metadata["syncFullHistory"] != true {
		t.Fatalf("expected syncFullHistory default true, got %#v", metadata["syncFullHistory"])
	}
}

func TestDefaultChannelMetadataKeepsExplicitFalseValues(t *testing.T) {
	metadata := defaultChannelMetadata(map[string]interface{}{
		"autoConnect":     false,
		"syncFullHistory": false,
	})

	if metadata["autoConnect"] != false {
		t.Fatalf("expected explicit autoConnect false to be preserved")
	}
	if metadata["syncFullHistory"] != false {
		t.Fatalf("expected explicit syncFullHistory false to be preserved")
	}
}
