package kingpin

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatTwoColumns(t *testing.T) {
	var buf bytes.Buffer
	FormatTwoColumns(&buf, 2, 2, 20, [][2]string{
		{"--hello", "Hello world help with something that is cool."},
	})
	expected := `  --hello  Hello
           world
           help with
           something
           that is
           cool.
`
	assert.Equal(t, expected, buf.String())
}

func TestFormatTwoColumnsWide(t *testing.T) {
	samples := [][2]string{
		{strings.Repeat("x", 29), "29 chars"},
		{strings.Repeat("x", 30), "30 chars"},
	}
	var buf bytes.Buffer
	FormatTwoColumns(&buf, 0, 0, 200, samples)
	expected := `xxxxxxxxxxxxxxxxxxxxxxxxxxxxx29 chars
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
                             30 chars
`
	assert.Equal(t, expected, buf.String())
}

func TestHiddenCommand(t *testing.T) {
	renderers := []struct {
		name     string
		renderer UsageRenderer
	}{
		{
			name:     "default",
			renderer: RenderDefault,
		},
		{
			name:     "compact",
			renderer: RenderCompact,
		},
		{
			name:     "long-help",
			renderer: RenderLongHelp,
		},
		{
			name:     "man-page",
			renderer: RenderManPage,
		},
	}

	for _, tp := range renderers {
		t.Run(tp.name, func(t *testing.T) {
			var buf bytes.Buffer
			a := New("test", "Test").UsageWriter(&buf).Terminate(nil)
			a.Command("visible", "visible")
			a.Command("hidden", "hidden").Hidden()

			a.UsageRenderer(tp.renderer)
			_, err := a.Parse(nil)
			require.Error(t, err)
			usage := buf.String()

			assert.NotContains(t, usage, "hidden")
			assert.Contains(t, usage, "visible")
		})
	}
}

func TestUsageRenderer(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test").UsageWriter(&buf).Terminate(nil)
	a.UsageRenderer(func(w io.Writer, ctx *UsageContext) error {
		_, err := fmt.Fprintf(w, "custom: %s", ctx.App.Name)
		return err
	})
	_, err := a.Parse([]string{"--help"})
	require.NoError(t, err)
	assert.Equal(t, "custom: test", buf.String())
}

func TestUsageTemplateFallback(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test").UsageWriter(&buf).Terminate(nil)
	a.UsageTemplate("{{ .App.Name }}")
	ctx, err := a.ParseContext(nil)
	require.NoError(t, err)
	err = a.UsageForContextWithTemplate(ctx, 2, "{{ .App.Name }}")
	require.NoError(t, err)
	assert.Equal(t, "test", buf.String())
}

func TestUsageFuncsApplyToDefaultHelp(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test help").UsageWriter(&buf).Terminate(nil)
	a.UsageFuncs(map[string]interface{}{
		"Wrap": func(indent int, s string) string { return "OVERRIDDEN\n" },
	})

	_, err := a.Parse([]string{"--help"})
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "OVERRIDDEN")
}

func TestUsageFuncsApplyToHiddenFlagPaths(t *testing.T) {
	for _, tp := range []struct {
		name     string
		flag     string
		contains string
	}{
		{
			name:     "help-long",
			flag:     "--help-long",
			contains: "OVERRIDDEN",
		},
		{
			name:     "help-man",
			flag:     "--help-man",
			contains: "OVERRIDDEN",
		},
	} {
		t.Run(tp.name, func(t *testing.T) {
			var buf bytes.Buffer
			a := New("test", "Test help").StdoutWriter(&buf).Terminate(nil)
			a.Flag("verbose", "Verbose output.").Short('v').Bool()
			a.UsageFuncs(map[string]interface{}{
				"Wrap": func(indent int, s string) string { return "OVERRIDDEN\n" },
				"Char": func(c rune) string { return "OVERRIDDEN" },
			})

			_, err := a.Parse([]string{tp.flag})
			require.NoError(t, err)

			assert.Contains(t, buf.String(), tp.contains)
		})
	}
}

func TestUsageDefaults(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test help").UsageWriter(&buf).Terminate(nil)

	_, err := a.Parse([]string{"--help"})
	require.NoError(t, err)

	usage := buf.String()
	assert.Contains(t, usage, "usage: test")
	assert.Contains(t, usage, "Test help")
	assert.Contains(t, usage, "--[no-]help")
}

func TestRendererPrioritizedWhenUsageFuncsSet(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test").UsageWriter(&buf).Terminate(nil)
	a.UsageFuncs(map[string]interface{}{
		"Wrap": func(indent int, s string) string { return "FROM_TEMPLATE\n" },
	})
	a.UsageRenderer(func(w io.Writer, ctx *UsageContext) error {
		_, err := fmt.Fprint(w, "FROM_RENDERER")
		return err
	})

	_, err := a.Parse([]string{"--help"})
	require.NoError(t, err)

	assert.Equal(t, "FROM_RENDERER", buf.String())
}

func TestCustomRendererDoesNotOverrideHiddenFlags(t *testing.T) {
	flags := []string{"--help-long", "--help-man", "--completion-script-bash", "--completion-script-zsh"}

	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			var buf bytes.Buffer
			a := New("test", "Test help").StdoutWriter(&buf).Terminate(nil)
			a.UsageRenderer(func(w io.Writer, ctx *UsageContext) error {
				_, err := fmt.Fprint(w, "CUSTOM_HELP")
				return err
			})

			_, err := a.Parse([]string{flag})
			require.NoError(t, err)

			assert.NotContains(t, buf.String(), "CUSTOM_HELP")
			assert.NotEmpty(t, buf.String())
		})
	}
}

func TestRendererAndUsageFuncsCombinedHiddenFlags(t *testing.T) {
	var helpBuf bytes.Buffer
	helpApp := New("test", "Test help").UsageWriter(&helpBuf).Terminate(nil)
	helpApp.UsageFuncs(map[string]interface{}{
		"Wrap": func(indent int, s string) string { return "FUNC_OVERRIDE\n" },
	})
	helpApp.UsageRenderer(func(w io.Writer, ctx *UsageContext) error {
		_, err := fmt.Fprint(w, "FROM_RENDERER")
		return err
	})

	_, err := helpApp.Parse([]string{"--help"})
	require.NoError(t, err)
	assert.Equal(t, "FROM_RENDERER", helpBuf.String())

	var longBuf bytes.Buffer
	longApp := New("test", "Test help").StdoutWriter(&longBuf).Terminate(nil)
	longApp.UsageFuncs(map[string]interface{}{
		"Wrap": func(indent int, s string) string { return "FUNC_OVERRIDE\n" },
	})
	longApp.UsageRenderer(func(w io.Writer, ctx *UsageContext) error {
		_, err := fmt.Fprint(w, "FROM_RENDERER")
		return err
	})

	_, err = longApp.Parse([]string{"--help-long"})
	require.NoError(t, err)
	assert.NotContains(t, longBuf.String(), "FROM_RENDERER")
	assert.Contains(t, longBuf.String(), "FUNC_OVERRIDE")
}

func TestUsageFuncsAloneUsesDefaultTemplate(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test help").UsageWriter(&buf).Terminate(nil)
	a.UsageFuncs(map[string]interface{}{
		"Wrap": func(indent int, s string) string { return "CUSTOM_WRAP\n" },
	})

	_, err := a.Parse([]string{"--help"})
	require.NoError(t, err)

	// UsageFuncs alone should fall back to the default template with custom usage functions.
	assert.Contains(t, buf.String(), "CUSTOM_WRAP")
	assert.Contains(t, buf.String(), "usage: test")
}

func TestUsageTemplateWinsOverRenderer(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test").UsageWriter(&buf).Terminate(nil)
	a.UsageTemplate("TEMPLATE:{{ .App.Name }}")
	a.UsageRenderer(func(w io.Writer, ctx *UsageContext) error {
		_, err := fmt.Fprint(w, "FROM_RENDERER")
		return err
	})

	_, err := a.Parse([]string{"--help"})
	require.NoError(t, err)

	assert.Equal(t, "TEMPLATE:test", buf.String())
}

func TestUsageForContextWithUsageFuncNilReturnsError(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test").UsageWriter(&buf).Terminate(nil)
	a.UsageRenderer(nil)

	ctx, err := a.ParseContext(nil)
	require.NoError(t, err)

	err = a.UsageForContextWithUsageRenderer(ctx, 2)
	require.EqualError(t, err, "no usage renderer provided")
}

func TestUsageTemplateWithUsageFuncsOverridesHelpers(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test").UsageWriter(&buf).Terminate(nil)
	a.UsageTemplate("{{ greet .App.Name }}")
	a.UsageFuncs(map[string]interface{}{
		"greet": func(name string) string { return "hello " + name },
	})

	_, err := a.Parse([]string{"--help"})
	require.NoError(t, err)

	assert.Equal(t, "hello test", buf.String())
}

func TestRenderersMatchTemplates(t *testing.T) {
	tests := []struct {
		name     string
		renderer UsageRenderer
		template string
	}{
		{
			name:     "default",
			renderer: RenderDefault,
			template: DefaultUsageTemplate,
		},
		{
			name:     "separate-optional-flags",
			renderer: RenderSeparateOptionalFlags,
			template: SeparateOptionalFlagsUsageTemplate,
		},
		{
			name:     "compact",
			renderer: RenderCompact,
			template: CompactUsageTemplate,
		},
		{
			name:     "long-help",
			renderer: RenderLongHelp,
			template: LongHelpTemplate,
		},
		{
			name:     "man-page",
			renderer: RenderManPage,
			template: ManPageTemplate,
		},
		{
			name:     "bash-completion",
			renderer: RenderBashCompletion,
			template: BashCompletionTemplate,
		},
		{
			name:     "zsh-completion",
			renderer: RenderZshCompletion,
			template: ZshCompletionTemplate,
		},
	}

	contexts := []struct {
		name string
		args []string
	}{
		{
			name: "root",
			args: nil,
		},
		{
			name: "nested-command",
			args: []string{"sub", "nested"},
		},
	}

	for _, test := range tests {
		for _, context := range contexts {
			t.Run(test.name+"/"+context.name, func(t *testing.T) {
				var templateBuf, rendererBuf bytes.Buffer
				makeApp := func(w *bytes.Buffer) *Application {
					a := New("test", "A test application.").UsageWriter(w).Terminate(nil).Version("1.0").Author("Test Author")
					a.Flag("verbose", "Enable verbose output.").Short('v').Bool()
					a.Flag("config", "Path to config file.").String()

					sub := a.Command("sub", "A subcommand.")
					sub.Flag("count", "Number of items.").Int()

					nested := sub.Command("nested", "A nested command.")
					nested.Arg("file", "File to process.").String()

					a.Command("other", "Another command.")
					return a
				}

				// Render with template.
				appT := makeApp(&templateBuf)
				ctx, err := appT.ParseContext(context.args)
				require.NoError(t, err)
				err = appT.UsageForContextWithTemplate(ctx, 2, test.template)
				require.NoError(t, err)

				// Render with programmatic renderer.
				appR := makeApp(&rendererBuf)
				ctx, err = appR.ParseContext(context.args)
				require.NoError(t, err)
				err = appR.usageForContextWithUsageRenderer(ctx, 2, test.renderer)
				require.NoError(t, err)

				// Validate that the output matches.
				assert.Equal(t, templateBuf.String(), rendererBuf.String())
			})
		}
	}
}

func TestArgEnvVar(t *testing.T) {
	var buf bytes.Buffer

	a := New("test", "Test").UsageWriter(&buf).Terminate(nil)
	a.Arg("arg", "Enable arg").Envar("ARG").String()
	a.Flag("flag", "Enable flag").Envar("FLAG").String()

	_, err := a.Parse([]string{"command", "--help"})
	require.NoError(t, err)
	usage := buf.String()
	assert.Contains(t, usage, "($ARG)")
	assert.Contains(t, usage, "($FLAG)")
}
