package resolver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeExecHelpers(t *testing.T) {
	dr := &DependencyResolver{}

	t.Run("ExecEnv_Nil", func(t *testing.T) {
		require.Nil(t, dr.encodeExecEnv(nil))
	})

	t.Run("ExecEnv_Encode", func(t *testing.T) {
		env := map[string]string{"KEY": "value"}
		enc := dr.encodeExecEnv(&env)
		require.NotNil(t, enc)
		require.Equal(t, "dmFsdWU=", (*enc)["KEY"])
	})

	t.Run("ExecOutputs", func(t *testing.T) {
		stderr := "err"
		stdout := "out"
		es, eo := dr.encodeExecOutputs(&stderr, &stdout)
		require.Equal(t, "ZXJy", *es)
		require.Equal(t, "b3V0", *eo)
	})

	t.Run("ExecOutputs_Nil", func(t *testing.T) {
		es, eo := dr.encodeExecOutputs(nil, nil)
		require.Nil(t, es)
		require.Nil(t, eo)
	})

	t.Run("EncodeStderr", func(t *testing.T) {
		txt := "oops"
		s := dr.encodeExecStderr(&txt)
		require.Contains(t, s, txt)
		require.Contains(t, s, "stderr = #\"\"\"")
	})

	t.Run("EncodeStderr_Nil", func(t *testing.T) {
		require.Equal(t, "    stderr = \"\"\n", dr.encodeExecStderr(nil))
	})

	t.Run("EncodeStdout", func(t *testing.T) {
		txt := "yay"
		s := dr.encodeExecStdout(&txt)
		require.Contains(t, s, txt)
		require.Contains(t, s, "stdout = #\"\"\"")
	})

	t.Run("EncodeStdout_Nil", func(t *testing.T) {
		require.Equal(t, "    stdout = \"\"\n", dr.encodeExecStdout(nil))
	})
}
