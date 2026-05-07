package kingpin

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatTwoColumns(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	FormatTwoColumns(buf, 2, 2, 20, [][2]string{
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
		{strings.Repeat("x", 30), "30 chars"}}
	buf := bytes.NewBuffer(nil)
	FormatTwoColumns(buf, 0, 0, 200, samples)
	expected := `xxxxxxxxxxxxxxxxxxxxxxxxxxxxx29 chars
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
                             30 chars
`
	assert.Equal(t, expected, buf.String())
}

func TestHiddenCommand(t *testing.T) {
	templates := []struct {
		name     string
		template string
		renderer UsageRenderer
	}{
		{
			name:     "default template",
			template: DefaultUsageTemplate,
		},
		{
			name:     "default renderer",
			renderer: RenderDefault,
		},
		{
			name:     "Compact template",
			template: CompactUsageTemplate,
		},
		{
			name:     "Compact renderer",
			renderer: RenderCompact,
		},
		{
			name:     "Long template",
			template: LongHelpTemplate,
		},
		{
			name:     "Long renderer",
			renderer: RenderLongHelp,
		},
		{
			name:     "Man template",
			template: ManPageTemplate,
		},
		{
			name:     "Man renderer",
			template: ManPageTemplate,
		},
	}

	for _, tp := range templates {
		t.Run(tp.name, func(t *testing.T) {
			var buf bytes.Buffer
			a := New("test", "Test").Writer(&buf).Terminate(nil)
			a.Command("visible", "visible")
			a.Command("hidden", "hidden").Hidden()
			if tp.template != "" {
				a.UsageTemplate(tp.template)
			}
			if tp.renderer != nil {
				a.UsageRenderer(tp.renderer)
			}
			_, err := a.Parse(nil)
			require.ErrorIs(t, err, ErrCommandNotSpecified)
			usage := buf.String()
			t.Logf("Usage for %s is:\n%s\n", tp.name, usage)

			assert.NotContains(t, usage, "hidden")
			assert.Contains(t, usage, "visible")
		})
	}
}

func TestUsageFuncs(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test").Writer(&buf).Terminate(nil)
	tpl := `{{ add 2 1 }}`
	a.UsageTemplate(tpl)
	a.UsageFuncs(template.FuncMap{
		"add": func(x, y int) int { return x + y },
	})
	_, err := a.Parse([]string{"--help"})
	require.NoError(t, err)
	usage := buf.String()
	assert.Equal(t, "3", usage)

	buf.Reset()
	a = New("test", "Test help").UsageWriter(&buf).Terminate(nil)
	a.UsageFuncs(map[string]interface{}{
		"Wrap": func(indent int, s string) string { return "OVERRIDDEN\n" },
	})

	_, err = a.Parse([]string{"--help"})
	require.NoError(t, err)
	usage = buf.String()
	assert.Contains(t, usage, "OVERRIDDEN")
}

func TestUsageFuncsApplyToHiddenFlagPaths(t *testing.T) {
	for _, tp := range []struct {
		name string
		flag string
	}{
		{
			name: "help-long",
			flag: "--help-long",
		},
		{
			name: "help-man",
			flag: "--help-man",
		},
	} {
		t.Run(tp.name, func(t *testing.T) {
			var buf bytes.Buffer
			a := New("test", "Test help").HiddenHelpWriter(&buf).Terminate(nil)
			a.Flag("verbose", "Verbose output.").Short('v').Bool()
			a.UsageFuncs(map[string]interface{}{
				"Wrap": func(indent int, s string) string { return "OVERRIDDEN\n" },
				"Char": func(c rune) string { return "OVERRIDDEN" },
			})

			_, err := a.Parse([]string{tp.flag})
			require.NoError(t, err)

			assert.Contains(t, buf.String(), "OVERRIDDEN")
		})
	}
}

func TestUsageRenderer(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test").Writer(&buf).Terminate(nil)
	a.UsageRenderer(func(w io.Writer, ctx *UsageContext) error {
		_, err := fmt.Fprintf(w, "custom: %s", ctx.App.Name)
		return err
	})

	_, err := a.Parse([]string{"--help"})
	require.NoError(t, err)
	usage := buf.String()
	assert.Equal(t, "custom: test", usage)
}

func TestCmdClause_HelpLong(t *testing.T) {
	var buf bytes.Buffer
	tpl := `{{define "FormatUsage"}}{{.HelpLong}}{{end -}}
{{template "FormatUsage" .Context.SelectedCommand}}`

	a := New("test", "Test").Writer(&buf).Terminate(nil)
	a.UsageTemplate(tpl)
	a.Command("command", "short help text").HelpLong("long help text")

	_, err := a.Parse([]string{"command", "--help"})
	require.NoError(t, err)
	usage := buf.String()
	assert.Equal(t, "long help text", usage)
}

func TestArgEnvVar(t *testing.T) {
	var buf bytes.Buffer

	a := New("test", "Test").Writer(&buf).Terminate(nil)
	a.Arg("arg", "Enable arg").Envar("ARG").String()
	a.Flag("flag", "Enable flag").Envar("FLAG").String()

	_, err := a.Parse([]string{"command", "--help"})
	require.NoError(t, err)
	usage := buf.String()
	assert.Contains(t, usage, "($ARG)")
	assert.Contains(t, usage, "($FLAG)")
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

func TestUsageRendererDoesNotOverrideHiddenFlags(t *testing.T) {
	flags := []string{"--help-long", "--help-man", "--completion-script-bash", "--completion-script-zsh", "--completion-script-fish"}

	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			var buf bytes.Buffer
			a := New("test", "Test help").HiddenHelpWriter(&buf).Terminate(nil)
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

func TestUsageTemplatePreferredOverUsageRenderer(t *testing.T) {
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
		{
			name:     "fish-completion",
			renderer: RenderFishCompletion,
			template: FishCompletionTemplate,
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

				appT := makeApp(&templateBuf)
				ctx, err := appT.ParseContext(context.args)
				require.NoError(t, err)
				err = appT.UsageForContextWithTemplate(ctx, 2, test.template)
				require.NoError(t, err)

				appR := makeApp(&rendererBuf)
				ctx, err = appR.ParseContext(context.args)
				require.NoError(t, err)
				err = appR.usageForContextWithUsageRenderer(ctx, 2, test.renderer)
				require.NoError(t, err)

				assert.Equal(t, templateBuf.String(), rendererBuf.String())
			})
		}
	}
}

func TestEnumFlagRendering(t *testing.T) {
	tests := []struct {
		name     string
		renderer UsageRenderer
		template string
	}{
		{
			name:     "default template",
			template: DefaultUsageTemplate,
		},
		{
			name:     "default renderer",
			renderer: RenderDefault,
		},
		{
			name:     "compact template",
			template: CompactUsageTemplate,
		},
		{
			name:     "compact renderer",
			renderer: RenderCompact,
		},
		{
			name:     "separate optional flags template",
			template: SeparateOptionalFlagsUsageTemplate,
		},
		{
			name:     "separate optional flags renderer",
			renderer: RenderSeparateOptionalFlags,
		},
		{
			name:     "long help template",
			template: LongHelpTemplate,
		},
		{
			name:     "long help renderer",
			renderer: RenderLongHelp,
		},
		{
			name:     "man page renderer",
			renderer: RenderManPage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			a := New("test", "Test enum flags").Writer(&buf).Terminate(nil)
			a.Flag("format", "Output format").Default("json").Enum("json", "yaml", "xml")
			a.Flag("level", "Log level").Enum("debug", "info", "warn", "error")

			if tt.template != "" {
				a.UsageTemplate(tt.template)
			}
			if tt.renderer != nil {
				a.UsageRenderer(tt.renderer)
			}

			_, err := a.Parse([]string{"--help"})
			require.NoError(t, err)
			usage := buf.String()

			assert.Contains(t, usage, "one of: json, yaml, xml")
			assert.Contains(t, usage, "one of: debug, info, warn, error")
		})
	}
}

func TestEnumArgRendering(t *testing.T) {
	tests := []struct {
		name     string
		renderer UsageRenderer
		template string
	}{
		{
			name:     "default template",
			template: DefaultUsageTemplate,
		},
		{
			name:     "default renderer",
			renderer: RenderDefault,
		},
		{
			name:     "compact template",
			template: CompactUsageTemplate,
		},
		{
			name:     "compact renderer",
			renderer: RenderCompact,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			a := New("test", "Test enum args").Writer(&buf).Terminate(nil)
			a.Arg("action", "Action to perform").Required().Enum("create", "read", "update", "delete")
			a.Arg("format", "Output format").Enum("json", "yaml", "xml")

			if tt.template != "" {
				a.UsageTemplate(tt.template)
			}
			if tt.renderer != nil {
				a.UsageRenderer(tt.renderer)
			}

			// Parse with a valid action to avoid required arg error
			ctx, err := a.ParseContext([]string{"create", "--help"})
			require.NoError(t, err)

			if tt.template != "" {
				err = a.UsageForContextWithTemplate(ctx, 2, tt.template)
			} else {
				err = a.usageForContextWithUsageRenderer(ctx, 2, tt.renderer)
			}
			require.NoError(t, err)
			usage := buf.String()

			assert.Contains(t, usage, "one of: create, read, update, delete")
			assert.Contains(t, usage, "one of: json, yaml, xml")
		})
	}
}

func TestEnumMultiValueRendering(t *testing.T) {
	tests := []struct {
		name     string
		renderer UsageRenderer
		template string
	}{
		{
			name:     "default template",
			template: DefaultUsageTemplate,
		},
		{
			name:     "default renderer",
			renderer: RenderDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			a := New("test", "Test multi-value enums").Writer(&buf).Terminate(nil)
			a.Flag("feature", "Enable features (repeatable)").Enums("auth", "logging", "metrics", "tracing")
			a.Arg("tags", "Tags to apply (repeatable)").Enums("dev", "prod", "staging", "test")

			if tt.template != "" {
				a.UsageTemplate(tt.template)
			}
			if tt.renderer != nil {
				a.UsageRenderer(tt.renderer)
			}

			_, err := a.Parse([]string{"--help"})
			require.NoError(t, err)
			usage := buf.String()

			// Enum values may wrap across lines in the output
			assert.Contains(t, usage, "auth")
			assert.Contains(t, usage, "logging")
			assert.Contains(t, usage, "metrics")
			assert.Contains(t, usage, "tracing")
			assert.Contains(t, usage, "dev")
			assert.Contains(t, usage, "prod")
			assert.Contains(t, usage, "staging")
			assert.Contains(t, usage, "test")
		})
	}
}

func TestEnumWithCommand(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test with commands").Writer(&buf).Terminate(nil)

	cmd := a.Command("deploy", "Deploy application")
	cmd.Flag("env", "Environment").Default("dev").Enum("dev", "staging", "prod")
	cmd.Arg("region", "AWS region").Required().Enum("us-east-1", "us-west-2", "eu-west-1")

	// Parse with valid region to avoid required arg error
	ctx, err := a.ParseContext([]string{"deploy", "us-east-1", "--help"})
	require.NoError(t, err)

	err = a.UsageForContext(ctx)
	require.NoError(t, err)
	usage := buf.String()

	assert.Contains(t, usage, "one of: dev, staging, prod")
	assert.Contains(t, usage, "one of: us-east-1, us-west-2, eu-west-1")
}

func TestEnumOptionsTemplateFunction(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test").Writer(&buf).Terminate(nil)
	a.Flag("format", "Output format").Enum("json", "yaml", "xml")

	// Use custom template that uses EnumOptions function
	tmpl := `{{ range .Context.Flags }}{{ if EnumOptions .Value }}Options: {{ range EnumOptions .Value }}{{ . }} {{ end }}{{ end }}{{ end }}`
	a.UsageTemplate(tmpl)

	_, err := a.Parse([]string{"--help"})
	require.NoError(t, err)
	usage := buf.String()

	assert.Contains(t, usage, "Options: json yaml xml")
}

func TestEnumWithEnvar(t *testing.T) {
	var buf bytes.Buffer
	a := New("test", "Test").Writer(&buf).Terminate(nil)
	a.Flag("format", "Output format").Envar("FORMAT").Enum("json", "yaml", "xml")
	a.Arg("level", "Log level").Envar("LEVEL").Enum("debug", "info", "warn")

	_, err := a.Parse([]string{"--help"})
	require.NoError(t, err)
	usage := buf.String()

	assert.Contains(t, usage, "$FORMAT")
	assert.Contains(t, usage, "one of: json, yaml, xml")
	assert.Contains(t, usage, "$LEVEL")
	assert.Contains(t, usage, "one of: debug, info, warn")
}

func TestFlagsToTwoColumnsWithEnums(t *testing.T) {
	flags := []*FlagModel{
		{
			Name:  "format",
			Help:  "Output format",
			Value: newEnumFlag(new(string), "json", "yaml", "xml"),
		},
		{
			Name:  "verbose",
			Help:  "Verbose output",
			Value: newBoolValue(new(bool)),
		},
	}

	rows := FlagsToTwoColumns(flags)
	require.Len(t, rows, 2)

	// First row should have enum values
	assert.Contains(t, rows[0][1], "Output format")
	assert.Contains(t, rows[0][1], "(one of: json, yaml, xml)")

	// Second row should not have enum values
	assert.Equal(t, "Verbose output", rows[1][1])
	assert.NotContains(t, rows[1][1], "(one of:")
}

func TestArgsToTwoColumnsWithEnums(t *testing.T) {
	args := []*ArgModel{
		{
			Name:     "action",
			Help:     "Action to perform",
			Required: true,
			Value:    newEnumFlag(new(string), "create", "delete", "update"),
		},
		{
			Name:     "file",
			Help:     "File to process",
			Required: false,
			Value:    newStringValue(new(string)),
		},
	}

	rows := ArgsToTwoColumns(args)
	require.Len(t, rows, 2)

	// First row should have enum values
	assert.Contains(t, rows[0][1], "Action to perform")
	assert.Contains(t, rows[0][1], "(one of: create, delete, update)")

	// Second row should not have enum values
	assert.Equal(t, "File to process", rows[1][1])
	assert.NotContains(t, rows[1][1], "(one of:")
}
