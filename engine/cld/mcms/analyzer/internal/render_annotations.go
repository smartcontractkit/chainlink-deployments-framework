package internal

// Rendering Annotation Constants
// These annotations control how entities are rendered in different formats.

const (
	// AnnotationRenderImportantName marks an entity as important.
	// When set, the renderer will highlight this entity (e.g., bold in HTML, ⭐ in text).
	// Type: boolean
	// Applies to: All analyzed entities (Proposal, BatchOperation, Call, Parameter)
	AnnotationRenderImportantName = "render.important"
	AnnotationRenderImportantType = "boolean"

	// AnnotationRenderEmojiName specifies an emoji to display alongside the entity.
	// Type: string (single emoji character)
	// Applies to: All analyzed entities
	AnnotationRenderEmojiName = "render.emoji"
	AnnotationRenderEmojiType = "string"

	// AnnotationRenderFormatterName specifies a custom formatter for the value.
	// Supported formatters:
	//   - "ethereum.address": formats as 0x-prefixed hex address
	//   - "ethereum.uint256": formats large numbers with commas
	//   - "hex": formats as hexadecimal
	//   - "truncate:<length>": truncates string to specified length
	// Type: string
	// Applies to: AnalyzedParameter
	AnnotationRenderFormatterName = "render.formatter"
	AnnotationRenderFormatterType = "string"

	// AnnotationRenderTemplateName specifies a custom template to use for rendering.
	// The value should be the template name (without format extension).
	// Format-specific templates will be loaded (e.g., "customCall.text.tmpl", "customCall.html.tmpl").
	// Type: string
	// Applies to: All analyzed entities
	AnnotationRenderTemplateName = "render.template"
	AnnotationRenderTemplateType = "string"

	// AnnotationRenderStyleName provides styling hints for the renderer.
	// Supported values: "bold", "italic", "underline", "code", "danger", "warning", "success", "info"
	// Type: string
	// Applies to: All analyzed entities
	AnnotationRenderStyleName = "render.style"
	AnnotationRenderStyleType = "string"

	// AnnotationRenderHideName indicates that the entity should be hidden in the output.
	// Type: boolean
	// Applies to: All analyzed entities
	AnnotationRenderHideName = "render.hide"
	AnnotationRenderHideType = "boolean"

	// AnnotationRenderExpandName controls whether nested entities are expanded by default.
	// Type: boolean
	// Applies to: Call, BatchOperation
	AnnotationRenderExpandName = "render.expand"
	AnnotationRenderExpandType = "boolean"

	// AnnotationRenderTooltipName provides tooltip/hover text for the entity.
	// Type: string
	// Applies to: All analyzed entities (mainly useful in HTML format)
	AnnotationRenderTooltipName = "render.tooltip"
	AnnotationRenderTooltipType = "string"
)

// Helper functions for creating rendering annotations

func ImportantAnnotation(important bool) annotation {
	return NewAnnotation(AnnotationRenderImportantName, AnnotationRenderImportantType, important)
}

func EmojiAnnotation(emoji string) annotation {
	return NewAnnotation(AnnotationRenderEmojiName, AnnotationRenderEmojiType, emoji)
}

func FormatterAnnotation(formatter string) annotation {
	return NewAnnotation(AnnotationRenderFormatterName, AnnotationRenderFormatterType, formatter)
}

func TemplateAnnotation(templateName string) annotation {
	return NewAnnotation(AnnotationRenderTemplateName, AnnotationRenderTemplateType, templateName)
}

func StyleAnnotation(style string) annotation {
	return NewAnnotation(AnnotationRenderStyleName, AnnotationRenderStyleType, style)
}

func HideAnnotation(hide bool) annotation {
	return NewAnnotation(AnnotationRenderHideName, AnnotationRenderHideType, hide)
}

func ExpandAnnotation(expand bool) annotation {
	return NewAnnotation(AnnotationRenderExpandName, AnnotationRenderExpandType, expand)
}

func TooltipAnnotation(tooltip string) annotation {
	return NewAnnotation(AnnotationRenderTooltipName, AnnotationRenderTooltipType, tooltip)
}
