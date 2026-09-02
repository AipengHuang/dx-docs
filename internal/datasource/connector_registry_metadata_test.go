package datasource

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetadataForTypesOnlyReturnsRegisteredConnectors(t *testing.T) {
	items := MetadataForTypes([]string{"rss", "notion", "unregistered"})
	require.Len(t, items, 2)
	require.Equal(t, "notion", items[0].Type)
	require.Equal(t, "rss", items[1].Type)
}
