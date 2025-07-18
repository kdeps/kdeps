package pklres_test

import (
	"net/url"
	"testing"

	"github.com/kdeps/kdeps/pkg/pklres"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestPklResourceReader(t *testing.T) {
	reader, err := pklres.InitializePklResource("test-graph", "", "", "", afero.NewMemMapFs())
	require.NoError(t, err)

	t.Run("Scheme", func(t *testing.T) {
		require.Equal(t, "pklres", reader.Scheme())
	})

	t.Run("IsGlobbable", func(t *testing.T) {
		require.False(t, reader.IsGlobbable())
	})

	t.Run("HasHierarchicalUris", func(t *testing.T) {
		require.False(t, reader.HasHierarchicalUris())
	})

	t.Run("ListElements", func(t *testing.T) {
		uri, _ := url.Parse("pklres:///test")
		elements, err := reader.ListElements(*uri)
		require.NoError(t, err)
		require.Nil(t, elements)
	})

	t.Run("Read_GetKeyValue", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("test-graph", "", "", "", afero.NewMemMapFs())
		require.NoError(t, err)

		// Set a value first
		setURI, _ := url.Parse("pklres://?op=set&collection=test1&key=testkey&value=testvalue")
		_, err = reader.Read(*setURI)
		require.NoError(t, err)

		// Get the value
		getURI, _ := url.Parse("pklres://?op=get&collection=test1&key=testkey")
		data, err := reader.Read(*getURI)
		require.NoError(t, err)
		require.Equal(t, `"testvalue"`, string(data)) // JSON format

		// Test with non-existent key
		getURI, _ = url.Parse("pklres://?op=get&collection=test1&key=nonexistent")
		_, err = reader.Read(*getURI)
		require.Error(t, err)
		require.Contains(t, err.Error(), "key 'nonexistent' not found")
	})

	t.Run("Read_SetKeyValue", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("test-graph", "", "", "", afero.NewMemMapFs())
		require.NoError(t, err)

		// Set a value
		uri, _ := url.Parse("pklres://?op=set&collection=test2&key=testkey&value=newvalue")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, `"newvalue"`, string(data)) // JSON format

		// Verify it was stored by getting it back
		getURI, _ := url.Parse("pklres://?op=get&collection=test2&key=testkey")
		data, err = reader.Read(*getURI)
		require.NoError(t, err)
		require.Equal(t, `"newvalue"`, string(data))

		// Test missing parameters
		uri, _ = url.Parse("pklres://?op=set")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "set operation requires collection and key parameters")

		uri, _ = url.Parse("pklres://?op=set&collection=test&key=testkey")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "set operation requires a value parameter")
	})

	t.Run("Read_ListKeys", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("test-graph", "", "", "", afero.NewMemMapFs())
		require.NoError(t, err)

		// Set multiple values in the same collection
		setURI1, _ := url.Parse("pklres://?op=set&collection=test3&key=key1&value=value1")
		_, err = reader.Read(*setURI1)
		require.NoError(t, err)

		setURI2, _ := url.Parse("pklres://?op=set&collection=test3&key=key2&value=value2")
		_, err = reader.Read(*setURI2)
		require.NoError(t, err)

		// List keys
		listURI, _ := url.Parse("pklres://?op=list&collection=test3")
		data, err := reader.Read(*listURI)
		require.NoError(t, err)
		require.Contains(t, string(data), "key1")
		require.Contains(t, string(data), "key2")

		// Test missing collection parameter
		listURI, _ = url.Parse("pklres://?op=list")
		_, err = reader.Read(*listURI)
		require.Error(t, err)
		require.Contains(t, err.Error(), "list operation requires collection parameter")
	})

	t.Run("Read_InvalidOperation", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("test-graph", "", "", "", afero.NewMemMapFs())
		require.NoError(t, err)

		// Test invalid operation
		uri, _ := url.Parse("pklres://?op=invalid")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported operation: invalid")
	})

	t.Run("Read_GetMissingParameters", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("test-graph", "", "", "", afero.NewMemMapFs())
		require.NoError(t, err)

		// Test get without collection
		uri, _ := url.Parse("pklres://?op=get&key=testkey")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "get operation requires collection and key parameters")

		// Test get without key
		uri, _ = url.Parse("pklres://?op=get&collection=test")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "get operation requires collection and key parameters")
	})

	t.Run("GraphIDScoping", func(t *testing.T) {
		reader1, err := pklres.InitializePklResource("graph1", "", "", "", afero.NewMemMapFs())
		require.NoError(t, err)

		reader2, err := pklres.InitializePklResource("graph2", "", "", "", afero.NewMemMapFs())
		require.NoError(t, err)

		// Set value in graph1
		setURI, _ := url.Parse("pklres://?op=set&collection=test&key=key&value=value1")
		_, err = reader1.Read(*setURI)
		require.NoError(t, err)

		// Set value in graph2
		setURI, _ = url.Parse("pklres://?op=set&collection=test&key=key&value=value2")
		_, err = reader2.Read(*setURI)
		require.NoError(t, err)

		// Get value from graph1
		getURI, _ := url.Parse("pklres://?op=get&collection=test&key=key")
		data, err := reader1.Read(*getURI)
		require.NoError(t, err)
		require.Equal(t, `"value1"`, string(data))

		// Get value from graph2
		data, err = reader2.Read(*getURI)
		require.NoError(t, err)
		require.Equal(t, `"value2"`, string(data))
	})

	t.Run("GlobalReader", func(t *testing.T) {
		// Test global reader functionality
		reader, err := pklres.InitializePklResource("test-graph", "", "", "", afero.NewMemMapFs())
		require.NoError(t, err)

		pklres.SetGlobalPklresReader(reader)
		defer pklres.SetGlobalPklresReader(nil)

		// Test that global reader is accessible
		globalReader := pklres.GetGlobalPklresReader()
		require.NotNil(t, globalReader)
		require.Equal(t, reader, globalReader)

		// Test context update
		err = pklres.UpdateGlobalPklresReaderContext("new-graph", "agent1", "1.0.0", "/path")
		require.NoError(t, err)

		require.Equal(t, "new-graph", globalReader.GraphID)
		require.Equal(t, "agent1", globalReader.CurrentAgent)
		require.Equal(t, "1.0.0", globalReader.CurrentVersion)
		require.Equal(t, "/path", globalReader.KdepsPath)
	})

	t.Run("NilReceiver", func(t *testing.T) {
		// Test that nil receiver uses global reader
		reader, err := pklres.InitializePklResource("test-graph", "", "", "", afero.NewMemMapFs())
		require.NoError(t, err)

		pklres.SetGlobalPklresReader(reader)
		defer pklres.SetGlobalPklresReader(nil)

		// Set a value using global reader
		setURI, _ := url.Parse("pklres://?op=set&collection=test&key=key&value=value")
		_, err = reader.Read(*setURI)
		require.NoError(t, err)

		// Get the value using nil receiver (should use global reader)
		var nilReader *pklres.PklResourceReader
		getURI, _ := url.Parse("pklres://?op=get&collection=test&key=key")
		data, err := nilReader.Read(*getURI)
		require.NoError(t, err)
		require.Equal(t, `"value"`, string(data))
	})
}
