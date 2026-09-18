package action

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/property"
)

func TestCanonicalizeCSVMetadata(t *testing.T) {
	cfg := &declcfg.DeclarativeConfig{Bundles: []declcfg.Bundle{{
		Properties: []property.Property{
			{Type: "other", Value: json.RawMessage(`{"preserved":true}`)},
			{
				Type: property.TypeCSVMetadata,
				Value: json.RawMessage(`{"apiServiceDefinitions":{"owned":[{"name":"v1.example.com","group":"example.com","version":"v1","kind":"Widget","deploymentName":"controller"}]},"crdDescriptions":{"owned":[{"name":"widgets.example.com","version":"v1","kind":"Widget","resources":[{"kind":"Service"}]}]}}`),
			},
		},
	}}}

	require.NoError(t, canonicalizeCSVMetadata(cfg))
	require.Equal(t, json.RawMessage(`{"preserved":true}`), cfg.Bundles[0].Properties[0].Value)
	require.NotContains(t, string(cfg.Bundles[0].Properties[1].Value), "deploymentName")
	require.NotContains(t, string(cfg.Bundles[0].Properties[1].Value), "resources")
}
