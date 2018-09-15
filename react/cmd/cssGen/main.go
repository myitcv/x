// cssGen is a temporary code generator for the myitcv.io/react.CSS type
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"text/template"
	"unicode/utf8"

	"myitcv.io/gogenerate"
)

// using
// https://github.com/Microsoft/TypeScript/blob/8b9fa4ce7420fdf2f540300dc80fa91f5b89ea93/lib/lib.dom.d.ts#L1692
// as a reference
// Used https://play.golang.org/p/uK-9pocUmVE to geerate.
//
var attrs = map[string]*typ{
	"AlignContent":                &typ{HTML: "align-content", React: "alignContent"},
	"AlignItems":                  &typ{HTML: "align-items", React: "alignItems"},
	"AlignSelf":                   &typ{HTML: "align-self", React: "alignSelf"},
	"AlignmentBaseline":           &typ{HTML: "alignment-baseline", React: "alignmentBaseline"},
	"All":                         &typ{HTML: "all", React: "all"},
	"Animation":                   &typ{HTML: "animation", React: "animation"},
	"AnimationDelay":              &typ{HTML: "animation-delay", React: "animationDelay"},
	"AnimationDirection":          &typ{HTML: "animation-direction", React: "animationDirection"},
	"AnimationDuration":           &typ{HTML: "animation-duration", React: "animationDuration"},
	"AnimationFillMode":           &typ{HTML: "animation-fill-mode", React: "animationFillMode"},
	"AnimationIterationCount":     &typ{HTML: "animation-iteration-count", React: "animationIterationCount"},
	"AnimationName":               &typ{HTML: "animation-name", React: "animationName"},
	"AnimationPlayState":          &typ{HTML: "animation-play-state", React: "animationPlayState"},
	"AnimationTimingFunction":     &typ{HTML: "animation-timing-function", React: "animationTimingFunction"},
	"Appearance":                  &typ{HTML: "appearance", React: "appearance"},
	"BackfaceVisibility":          &typ{HTML: "backface-visibility", React: "backfaceVisibility"},
	"Background":                  &typ{HTML: "background", React: "background"},
	"BackgroundAttachment":        &typ{HTML: "background-attachment", React: "backgroundAttachment"},
	"BackgroundClip":              &typ{HTML: "background-clip", React: "backgroundClip"},
	"BackgroundColor":             &typ{HTML: "background-color", React: "backgroundColor"},
	"BackgroundImage":             &typ{HTML: "background-image", React: "backgroundImage"},
	"BackgroundOrigin":            &typ{HTML: "background-origin", React: "backgroundOrigin"},
	"BackgroundPosition":          &typ{HTML: "background-position", React: "backgroundPosition"},
	"BackgroundRepeat":            &typ{HTML: "background-repeat", React: "backgroundRepeat"},
	"BackgroundSize":              &typ{HTML: "background-size", React: "backgroundSize"},
	"BaselineShift":               &typ{HTML: "baseline-shift", React: "baselineShift"},
	"Binding":                     &typ{HTML: "binding", React: "binding"},
	"Bleed":                       &typ{HTML: "bleed", React: "bleed"},
	"BookmarkLabel":               &typ{HTML: "bookmark-label", React: "bookmarkLabel"},
	"BookmarkLevel":               &typ{HTML: "bookmark-level", React: "bookmarkLevel"},
	"BookmarkState":               &typ{HTML: "bookmark-state", React: "bookmarkState"},
	"Border":                      &typ{HTML: "border", React: "border"},
	"BorderBottom":                &typ{HTML: "border-bottom", React: "borderBottom"},
	"BorderBottomColor":           &typ{HTML: "border-bottom-color", React: "borderBottomColor"},
	"BorderBottomLeftRadius":      &typ{HTML: "border-bottom-left-radius", React: "borderBottomLeftRadius"},
	"BorderBottomRightRadius":     &typ{HTML: "border-bottom-right-radius", React: "borderBottomRightRadius"},
	"BorderBottomStyle":           &typ{HTML: "border-bottom-style", React: "borderBottomStyle"},
	"BorderBottomWidth":           &typ{HTML: "border-bottom-width", React: "borderBottomWidth"},
	"BorderBoundary":              &typ{HTML: "border-boundary", React: "borderBoundary"},
	"BorderCollapse":              &typ{HTML: "border-collapse", React: "borderCollapse"},
	"BorderColor":                 &typ{HTML: "border-color", React: "borderColor"},
	"BorderImage":                 &typ{HTML: "border-image", React: "borderImage"},
	"BorderImageOutset":           &typ{HTML: "border-image-outset", React: "borderImageOutset"},
	"BorderImageRepeat":           &typ{HTML: "border-image-repeat", React: "borderImageRepeat"},
	"BorderImageSlice":            &typ{HTML: "border-image-slice", React: "borderImageSlice"},
	"BorderImageSource":           &typ{HTML: "border-image-source", React: "borderImageSource"},
	"BorderImageWidth":            &typ{HTML: "border-image-width", React: "borderImageWidth"},
	"BorderLeft":                  &typ{HTML: "border-left", React: "borderLeft"},
	"BorderLeftColor":             &typ{HTML: "border-left-color", React: "borderLeftColor"},
	"BorderLeftStyle":             &typ{HTML: "border-left-style", React: "borderLeftStyle"},
	"BorderLeftWidth":             &typ{HTML: "border-left-width", React: "borderLeftWidth"},
	"BorderRadius":                &typ{HTML: "border-radius", React: "borderRadius"},
	"BorderRight":                 &typ{HTML: "border-right", React: "borderRight"},
	"BorderRightColor":            &typ{HTML: "border-right-color", React: "borderRightColor"},
	"BorderRightStyle":            &typ{HTML: "border-right-style", React: "borderRightStyle"},
	"BorderRightWidth":            &typ{HTML: "border-right-width", React: "borderRightWidth"},
	"BorderSpacing":               &typ{HTML: "border-spacing", React: "borderSpacing"},
	"BorderStyle":                 &typ{HTML: "border-style", React: "borderStyle"},
	"BorderTop":                   &typ{HTML: "border-top", React: "borderTop"},
	"BorderTopColor":              &typ{HTML: "border-top-color", React: "borderTopColor"},
	"BorderTopLeftRadius":         &typ{HTML: "border-top-left-radius", React: "borderTopLeftRadius"},
	"BorderTopRightRadius":        &typ{HTML: "border-top-right-radius", React: "borderTopRightRadius"},
	"BorderTopStyle":              &typ{HTML: "border-top-style", React: "borderTopStyle"},
	"BorderTopWidth":              &typ{HTML: "border-top-width", React: "borderTopWidth"},
	"BorderWidth":                 &typ{HTML: "border-width", React: "borderWidth"},
	"Bottom":                      &typ{HTML: "bottom", React: "bottom"},
	"BoxDecorationBreak":          &typ{HTML: "box-decoration-break", React: "boxDecorationBreak"},
	"BoxShadow":                   &typ{HTML: "box-shadow", React: "boxShadow"},
	"BoxSizing":                   &typ{HTML: "box-sizing", React: "boxSizing"},
	"BoxSnap":                     &typ{HTML: "box-snap", React: "boxSnap"},
	"BoxSuppress":                 &typ{HTML: "box-suppress", React: "boxSuppress"},
	"BreakAfter":                  &typ{HTML: "break-after", React: "breakAfter"},
	"BreakBefore":                 &typ{HTML: "break-before", React: "breakBefore"},
	"BreakInside":                 &typ{HTML: "break-inside", React: "breakInside"},
	"CaptionSide":                 &typ{HTML: "caption-side", React: "captionSide"},
	"Caret":                       &typ{HTML: "caret", React: "caret"},
	"CaretShape":                  &typ{HTML: "caret-shape", React: "caretShape"},
	"Chains":                      &typ{HTML: "chains", React: "chains"},
	"Clear":                       &typ{HTML: "clear", React: "clear"},
	"ClipPath":                    &typ{HTML: "clip-path", React: "clipPath"},
	"ClipRule":                    &typ{HTML: "clip-rule", React: "clipRule"},
	"Color":                       &typ{HTML: "color", React: "color"},
	"ColorInterpolationFilters":   &typ{HTML: "color-interpolation-filters", React: "colorInterpolationFilters"},
	"ColumnCount":                 &typ{HTML: "column-count", React: "columnCount"},
	"ColumnFill":                  &typ{HTML: "column-fill", React: "columnFill"},
	"ColumnGap":                   &typ{HTML: "column-gap", React: "columnGap"},
	"ColumnRule":                  &typ{HTML: "column-rule", React: "columnRule"},
	"ColumnRuleColor":             &typ{HTML: "column-rule-color", React: "columnRuleColor"},
	"ColumnRuleStyle":             &typ{HTML: "column-rule-style", React: "columnRuleStyle"},
	"ColumnRuleWidth":             &typ{HTML: "column-rule-width", React: "columnRuleWidth"},
	"ColumnSpan":                  &typ{HTML: "column-span", React: "columnSpan"},
	"ColumnWidth":                 &typ{HTML: "column-width", React: "columnWidth"},
	"Columns":                     &typ{HTML: "columns", React: "columns"},
	"Contain":                     &typ{HTML: "contain", React: "contain"},
	"Content":                     &typ{HTML: "content", React: "content"},
	"CounterIncrement":            &typ{HTML: "counter-increment", React: "counterIncrement"},
	"CounterReset":                &typ{HTML: "counter-reset", React: "counterReset"},
	"CounterSet":                  &typ{HTML: "counter-set", React: "counterSet"},
	"Crop":                        &typ{HTML: "crop", React: "crop"},
	"Cue":                         &typ{HTML: "cue", React: "cue"},
	"CueAfter":                    &typ{HTML: "cue-after", React: "cueAfter"},
	"CueBefore":                   &typ{HTML: "cue-before", React: "cueBefore"},
	"Cursor":                      &typ{HTML: "cursor", React: "cursor"},
	"Direction":                   &typ{HTML: "direction", React: "direction"},
	"Display":                     &typ{HTML: "display", React: "display"},
	"DisplayInside":               &typ{HTML: "display-inside", React: "displayInside"},
	"DisplayList":                 &typ{HTML: "display-list", React: "displayList"},
	"DisplayOutside":              &typ{HTML: "display-outside", React: "displayOutside"},
	"DominantBaseline":            &typ{HTML: "dominant-baseline", React: "dominantBaseline"},
	"EmptyCells":                  &typ{HTML: "empty-cells", React: "emptyCells"},
	"Filter":                      &typ{HTML: "filter", React: "filter"},
	"Flex":                        &typ{HTML: "flex", React: "flex"},
	"FlexBasis":                   &typ{HTML: "flex-basis", React: "flexBasis"},
	"FlexDirection":               &typ{HTML: "flex-direction", React: "flexDirection"},
	"FlexFlow":                    &typ{HTML: "flex-flow", React: "flexFlow"},
	"FlexGrow":                    &typ{HTML: "flex-grow", React: "flexGrow"},
	"FlexShrink":                  &typ{HTML: "flex-shrink", React: "flexShrink"},
	"FlexWrap":                    &typ{HTML: "flex-wrap", React: "flexWrap"},
	"Float":                       &typ{HTML: "float", React: "float"},
	"FloatOffset":                 &typ{HTML: "float-offset", React: "floatOffset"},
	"FloodColor":                  &typ{HTML: "flood-color", React: "floodColor"},
	"FloodOpacity":                &typ{HTML: "flood-opacity", React: "floodOpacity"},
	"FlowFrom":                    &typ{HTML: "flow-from", React: "flowFrom"},
	"FlowInto":                    &typ{HTML: "flow-into", React: "flowInto"},
	"Font":                        &typ{HTML: "font", React: "font"},
	"FontFamily":                  &typ{HTML: "font-family", React: "fontFamily"},
	"FontFeatureSettings":         &typ{HTML: "font-feature-settings", React: "fontFeatureSettings"},
	"FontKerning":                 &typ{HTML: "font-kerning", React: "fontKerning"},
	"FontLanguageOverride":        &typ{HTML: "font-language-override", React: "fontLanguageOverride"},
	"FontMaxSize":                 &typ{HTML: "font-max-size", React: "fontMaxSize"},
	"FontMinSize":                 &typ{HTML: "font-min-size", React: "fontMinSize"},
	"FontOpticalSizing":           &typ{HTML: "font-optical-sizing", React: "fontOpticalSizing"},
	"FontPalette":                 &typ{HTML: "font-palette", React: "fontPalette"},
	"FontPresentation":            &typ{HTML: "font-presentation", React: "fontPresentation"},
	"FontSize":                    &typ{HTML: "font-size", React: "fontSize"},
	"FontSizeAdjust":              &typ{HTML: "font-size-adjust", React: "fontSizeAdjust"},
	"FontStretch":                 &typ{HTML: "font-stretch", React: "fontStretch"},
	"FontStyle":                   &typ{HTML: "font-style", React: "fontStyle"},
	"FontSynthesis":               &typ{HTML: "font-synthesis", React: "fontSynthesis"},
	"FontVariant":                 &typ{HTML: "font-variant", React: "fontVariant"},
	"FontVariantAlternates":       &typ{HTML: "font-variant-alternates", React: "fontVariantAlternates"},
	"FontVariantCaps":             &typ{HTML: "font-variant-caps", React: "fontVariantCaps"},
	"FontVariantEastAsian":        &typ{HTML: "font-variant-east-asian", React: "fontVariantEastAsian"},
	"FontVariantLigatures":        &typ{HTML: "font-variant-ligatures", React: "fontVariantLigatures"},
	"FontVariantNumeric":          &typ{HTML: "font-variant-numeric", React: "fontVariantNumeric"},
	"FontVariantPosition":         &typ{HTML: "font-variant-position", React: "fontVariantPosition"},
	"FontVariationSettings":       &typ{HTML: "font-variation-settings", React: "fontVariationSettings"},
	"FontWeight":                  &typ{HTML: "font-weight", React: "fontWeight"},
	"Grid":                        &typ{HTML: "grid", React: "grid"},
	"GridArea":                    &typ{HTML: "grid-area", React: "gridArea"},
	"GridAutoColumns":             &typ{HTML: "grid-auto-columns", React: "gridAutoColumns"},
	"GridAutoFlow":                &typ{HTML: "grid-auto-flow", React: "gridAutoFlow"},
	"GridAutoRows":                &typ{HTML: "grid-auto-rows", React: "gridAutoRows"},
	"GridColumn":                  &typ{HTML: "grid-column", React: "gridColumn"},
	"GridColumnEnd":               &typ{HTML: "grid-column-end", React: "gridColumnEnd"},
	"GridColumnStart":             &typ{HTML: "grid-column-start", React: "gridColumnStart"},
	"GridRow":                     &typ{HTML: "grid-row", React: "gridRow"},
	"GridRowEnd":                  &typ{HTML: "grid-row-end", React: "gridRowEnd"},
	"GridRowStart":                &typ{HTML: "grid-row-start", React: "gridRowStart"},
	"GridTemplate":                &typ{HTML: "grid-template", React: "gridTemplate"},
	"GridTemplateAreas":           &typ{HTML: "grid-template-areas", React: "gridTemplateAreas"},
	"GridTemplateColumns":         &typ{HTML: "grid-template-columns", React: "gridTemplateColumns"},
	"GridTemplateRows":            &typ{HTML: "grid-template-rows", React: "gridTemplateRows"},
	"HangingPunctuation":          &typ{HTML: "hanging-punctuation", React: "hangingPunctuation"},
	"Height":                      &typ{HTML: "height", React: "height"},
	"Hyphens":                     &typ{HTML: "hyphens", React: "hyphens"},
	"Icon":                        &typ{HTML: "icon", React: "icon"},
	"ImageOrientation":            &typ{HTML: "image-orientation", React: "imageOrientation"},
	"ImageRendering":              &typ{HTML: "image-rendering", React: "imageRendering"},
	"ImageResolution":             &typ{HTML: "image-resolution", React: "imageResolution"},
	"ImeMode":                     &typ{HTML: "ime-mode", React: "imeMode"},
	"InitialLetters":              &typ{HTML: "initial-letters", React: "initialLetters"},
	"InitialLettersAlign":         &typ{HTML: "initial-letters-align", React: "initialLettersAlign"},
	"InitialLettersWrap":          &typ{HTML: "initial-letters-wrap", React: "initialLettersWrap"},
	"InlineSizing":                &typ{HTML: "inline-sizing", React: "inlineSizing"},
	"JustifyContent":              &typ{HTML: "justify-content", React: "justifyContent"},
	"JustifyItems":                &typ{HTML: "justify-items", React: "justifyItems"},
	"JustifySelf":                 &typ{HTML: "justify-self", React: "justifySelf"},
	"Left":                        &typ{HTML: "left", React: "left"},
	"LetterSpacing":               &typ{HTML: "letter-spacing", React: "letterSpacing"},
	"LightingColor":               &typ{HTML: "lighting-color", React: "lightingColor"},
	"LineBreak":                   &typ{HTML: "line-break", React: "lineBreak"},
	"LineGrid":                    &typ{HTML: "line-grid", React: "lineGrid"},
	"LineHeight":                  &typ{HTML: "line-height", React: "lineHeight"},
	"LineSnap":                    &typ{HTML: "line-snap", React: "lineSnap"},
	"ListStyle":                   &typ{HTML: "list-style", React: "listStyle"},
	"ListStyleImage":              &typ{HTML: "list-style-image", React: "listStyleImage"},
	"ListStylePosition":           &typ{HTML: "list-style-position", React: "listStylePosition"},
	"ListStyleType":               &typ{HTML: "list-style-type", React: "listStyleType"},
	"Margin":                      &typ{HTML: "margin", React: "margin"},
	"MarginBottom":                &typ{HTML: "margin-bottom", React: "marginBottom"},
	"MarginLeft":                  &typ{HTML: "margin-left", React: "marginLeft"},
	"MarginRight":                 &typ{HTML: "margin-right", React: "marginRight"},
	"MarginTop":                   &typ{HTML: "margin-top", React: "marginTop"},
	"MarkerSide":                  &typ{HTML: "marker-side", React: "markerSide"},
	"Marks":                       &typ{HTML: "marks", React: "marks"},
	"Mask":                        &typ{HTML: "mask", React: "mask"},
	"MaskBox":                     &typ{HTML: "mask-box", React: "maskBox"},
	"MaskBoxOutset":               &typ{HTML: "mask-box-outset", React: "maskBoxOutset"},
	"MaskBoxRepeat":               &typ{HTML: "mask-box-repeat", React: "maskBoxRepeat"},
	"MaskBoxSlice":                &typ{HTML: "mask-box-slice", React: "maskBoxSlice"},
	"MaskBoxSource":               &typ{HTML: "mask-box-source", React: "maskBoxSource"},
	"MaskBoxWidth":                &typ{HTML: "mask-box-width", React: "maskBoxWidth"},
	"MaskClip":                    &typ{HTML: "mask-clip", React: "maskClip"},
	"MaskImage":                   &typ{HTML: "mask-image", React: "maskImage"},
	"MaskOrigin":                  &typ{HTML: "mask-origin", React: "maskOrigin"},
	"MaskPosition":                &typ{HTML: "mask-position", React: "maskPosition"},
	"MaskRepeat":                  &typ{HTML: "mask-repeat", React: "maskRepeat"},
	"MaskSize":                    &typ{HTML: "mask-size", React: "maskSize"},
	"MaskSourceType":              &typ{HTML: "mask-source-type", React: "maskSourceType"},
	"MaskType":                    &typ{HTML: "mask-type", React: "maskType"},
	"MaxHeight":                   &typ{HTML: "max-height", React: "maxHeight"},
	"MaxLines":                    &typ{HTML: "max-lines", React: "maxLines"},
	"MaxWidth":                    &typ{HTML: "max-width", React: "maxWidth"},
	"MinHeight":                   &typ{HTML: "min-height", React: "minHeight"},
	"MinWidth":                    &typ{HTML: "min-width", React: "minWidth"},
	"MoveTo":                      &typ{HTML: "move-to", React: "moveTo"},
	"NavDown":                     &typ{HTML: "nav-down", React: "navDown"},
	"NavIndex":                    &typ{HTML: "nav-index", React: "navIndex"},
	"NavLeft":                     &typ{HTML: "nav-left", React: "navLeft"},
	"NavRight":                    &typ{HTML: "nav-right", React: "navRight"},
	"NavUp":                       &typ{HTML: "nav-up", React: "navUp"},
	"ObjectFit":                   &typ{HTML: "object-fit", React: "objectFit"},
	"ObjectPosition":              &typ{HTML: "object-position", React: "objectPosition"},
	"Opacity":                     &typ{HTML: "opacity", React: "opacity"},
	"Order":                       &typ{HTML: "order", React: "order"},
	"Orphans":                     &typ{HTML: "orphans", React: "orphans"},
	"Outline":                     &typ{HTML: "outline", React: "outline"},
	"OutlineColor":                &typ{HTML: "outline-color", React: "outlineColor"},
	"OutlineOffset":               &typ{HTML: "outline-offset", React: "outlineOffset"},
	"OutlineStyle":                &typ{HTML: "outline-style", React: "outlineStyle"},
	"OutlineWidth":                &typ{HTML: "outline-width", React: "outlineWidth"},
	"Overflow":                    &typ{HTML: "overflow", React: "overflow"},
	"OverflowWrap":                &typ{HTML: "overflow-wrap", React: "overflowWrap"},
	"OverflowX":                   &typ{HTML: "overflow-x", React: "overflowX"},
	"OverflowY":                   &typ{HTML: "overflow-y", React: "overflowY"},
	"Padding":                     &typ{HTML: "padding", React: "padding"},
	"PaddingBottom":               &typ{HTML: "padding-bottom", React: "paddingBottom"},
	"PaddingLeft":                 &typ{HTML: "padding-left", React: "paddingLeft"},
	"PaddingRight":                &typ{HTML: "padding-right", React: "paddingRight"},
	"PaddingTop":                  &typ{HTML: "padding-top", React: "paddingTop"},
	"Page":                        &typ{HTML: "page", React: "page"},
	"PageBreakAfter":              &typ{HTML: "page-break-after", React: "pageBreakAfter"},
	"PageBreakBefore":             &typ{HTML: "page-break-before", React: "pageBreakBefore"},
	"PageBreakInside":             &typ{HTML: "page-break-inside", React: "pageBreakInside"},
	"PagePolicy":                  &typ{HTML: "page-policy", React: "pagePolicy"},
	"Pause":                       &typ{HTML: "pause", React: "pause"},
	"PauseAfter":                  &typ{HTML: "pause-after", React: "pauseAfter"},
	"PauseBefore":                 &typ{HTML: "pause-before", React: "pauseBefore"},
	"Perspective":                 &typ{HTML: "perspective", React: "perspective"},
	"PerspectiveOrigin":           &typ{HTML: "perspective-origin", React: "perspectiveOrigin"},
	"PolarAnchor":                 &typ{HTML: "polar-anchor", React: "polarAnchor"},
	"PolarAngle":                  &typ{HTML: "polar-angle", React: "polarAngle"},
	"PolarDistance":               &typ{HTML: "polar-distance", React: "polarDistance"},
	"PolarOrigin":                 &typ{HTML: "polar-origin", React: "polarOrigin"},
	"Position":                    &typ{HTML: "position", React: "position"},
	"PresentationLevel":           &typ{HTML: "presentation-level", React: "presentationLevel"},
	"Quotes":                      &typ{HTML: "quotes", React: "quotes"},
	"RegionFragment":              &typ{HTML: "region-fragment", React: "regionFragment"},
	"Resize":                      &typ{HTML: "resize", React: "resize"},
	"Rest":                        &typ{HTML: "rest", React: "rest"},
	"RestAfter":                   &typ{HTML: "rest-after", React: "restAfter"},
	"RestBefore":                  &typ{HTML: "rest-before", React: "restBefore"},
	"Right":                       &typ{HTML: "right", React: "right"},
	"Rotation":                    &typ{HTML: "rotation", React: "rotation"},
	"RotationPoint":               &typ{HTML: "rotation-point", React: "rotationPoint"},
	"RowGap":                      &typ{HTML: "row-gap", React: "rowGap"},
	"RubyAlign":                   &typ{HTML: "ruby-align", React: "rubyAlign"},
	"RubyMerge":                   &typ{HTML: "ruby-merge", React: "rubyMerge"},
	"RubyPosition":                &typ{HTML: "ruby-position", React: "rubyPosition"},
	"ScrollPadding":               &typ{HTML: "scroll-padding", React: "scrollPadding"},
	"ScrollPaddingBlock":          &typ{HTML: "scroll-padding-block", React: "scrollPaddingBlock"},
	"ScrollPaddingBlockEnd":       &typ{HTML: "scroll-padding-block-end", React: "scrollPaddingBlockEnd"},
	"ScrollPaddingBlockStart":     &typ{HTML: "scroll-padding-block-start", React: "scrollPaddingBlockStart"},
	"ScrollPaddingBottom":         &typ{HTML: "scroll-padding-bottom", React: "scrollPaddingBottom"},
	"ScrollPaddingInline":         &typ{HTML: "scroll-padding-inline", React: "scrollPaddingInline"},
	"ScrollPaddingInlineEnd":      &typ{HTML: "scroll-padding-inline-end", React: "scrollPaddingInlineEnd"},
	"ScrollPaddingInlineStart":    &typ{HTML: "scroll-padding-inline-start", React: "scrollPaddingInlineStart"},
	"ScrollPaddingLeft":           &typ{HTML: "scroll-padding-left", React: "scrollPaddingLeft"},
	"ScrollPaddingRight":          &typ{HTML: "scroll-padding-right", React: "scrollPaddingRight"},
	"ScrollPaddingTop":            &typ{HTML: "scroll-padding-top", React: "scrollPaddingTop"},
	"ScrollSnapAlign":             &typ{HTML: "scroll-snap-align", React: "scrollSnapAlign"},
	"ScrollSnapMargin":            &typ{HTML: "scroll-snap-margin", React: "scrollSnapMargin"},
	"ScrollSnapMarginBlock":       &typ{HTML: "scroll-snap-margin-block", React: "scrollSnapMarginBlock"},
	"ScrollSnapMarginBlockEnd":    &typ{HTML: "scroll-snap-margin-block-end", React: "scrollSnapMarginBlockEnd"},
	"ScrollSnapMarginBlockStart":  &typ{HTML: "scroll-snap-margin-block-start", React: "scrollSnapMarginBlockStart"},
	"ScrollSnapMarginBottom":      &typ{HTML: "scroll-snap-margin-bottom", React: "scrollSnapMarginBottom"},
	"ScrollSnapMarginInline":      &typ{HTML: "scroll-snap-margin-inline", React: "scrollSnapMarginInline"},
	"ScrollSnapMarginInlineEnd":   &typ{HTML: "scroll-snap-margin-inline-end", React: "scrollSnapMarginInlineEnd"},
	"ScrollSnapMarginInlineStart": &typ{HTML: "scroll-snap-margin-inline-start", React: "scrollSnapMarginInlineStart"},
	"ScrollSnapMarginLeft":        &typ{HTML: "scroll-snap-margin-left", React: "scrollSnapMarginLeft"},
	"ScrollSnapMarginRight":       &typ{HTML: "scroll-snap-margin-right", React: "scrollSnapMarginRight"},
	"ScrollSnapMarginTop":         &typ{HTML: "scroll-snap-margin-top", React: "scrollSnapMarginTop"},
	"ScrollSnapStop":              &typ{HTML: "scroll-snap-stop", React: "scrollSnapStop"},
	"ScrollSnapType":              &typ{HTML: "scroll-snap-type", React: "scrollSnapType"},
	"ShapeImageThreshold":         &typ{HTML: "shape-image-threshold", React: "shapeImageThreshold"},
	"ShapeInside":                 &typ{HTML: "shape-inside", React: "shapeInside"},
	"ShapeOutside":                &typ{HTML: "shape-outside", React: "shapeOutside"},
	"ShapeMargin":                 &typ{HTML: "shape-margin", React: "shapeMargin"},
	"Size":                        &typ{HTML: "size", React: "size"},
	"Speak":                       &typ{HTML: "speak", React: "speak"},
	"SpeakAs":                     &typ{HTML: "speak-as", React: "speakAs"},
	"StringSet":                   &typ{HTML: "string-set", React: "stringSet"},
	"TabSize":                     &typ{HTML: "tab-size", React: "tabSize"},
	"TableLayout":                 &typ{HTML: "table-layout", React: "tableLayout"},
	"TextAlign":                   &typ{HTML: "text-align", React: "textAlign"},
	"TextAlignLast":               &typ{HTML: "text-align-last", React: "textAlignLast"},
	"TextCombineUpright":          &typ{HTML: "text-combine-upright", React: "textCombineUpright"},
	"TextDecoration":              &typ{HTML: "text-decoration", React: "textDecoration"},
	"TextDecorationColor":         &typ{HTML: "text-decoration-color", React: "textDecorationColor"},
	"TextDecorationLine":          &typ{HTML: "text-decoration-line", React: "textDecorationLine"},
	"TextDecorationSkip":          &typ{HTML: "text-decoration-skip", React: "textDecorationSkip"},
	"TextDecorationStyle":         &typ{HTML: "text-decoration-style", React: "textDecorationStyle"},
	"TextEmphasis":                &typ{HTML: "text-emphasis", React: "textEmphasis"},
	"TextEmphasisColor":           &typ{HTML: "text-emphasis-color", React: "textEmphasisColor"},
	"TextEmphasisPosition":        &typ{HTML: "text-emphasis-position", React: "textEmphasisPosition"},
	"TextEmphasisStyle":           &typ{HTML: "text-emphasis-style", React: "textEmphasisStyle"},
	"TextIndent":                  &typ{HTML: "text-indent", React: "textIndent"},
	"TextJustify":                 &typ{HTML: "text-justify", React: "textJustify"},
	"TextOrientation":             &typ{HTML: "text-orientation", React: "textOrientation"},
	"TextOverflow":                &typ{HTML: "text-overflow", React: "textOverflow"},
	"TextShadow":                  &typ{HTML: "text-shadow", React: "textShadow"},
	"TextSpaceCollapse":           &typ{HTML: "text-space-collapse", React: "textSpaceCollapse"},
	"TextTransform":               &typ{HTML: "text-transform", React: "textTransform"},
	"TextUnderlinePosition":       &typ{HTML: "text-underline-position", React: "textUnderlinePosition"},
	"TextWrap":                    &typ{HTML: "text-wrap", React: "textWrap"},
	"TouchAction":                 &typ{HTML: "touch-action", React: "touchAction"},
	"Top":                         &typ{HTML: "top", React: "top"},
	"Transform":                   &typ{HTML: "transform", React: "transform"},
	"TransformOrigin":             &typ{HTML: "transform-origin", React: "transformOrigin"},
	"TransformStyle":              &typ{HTML: "transform-style", React: "transformStyle"},
	"Transition":                  &typ{HTML: "transition", React: "transition"},
	"TransitionDelay":             &typ{HTML: "transition-delay", React: "transitionDelay"},
	"TransitionDuration":          &typ{HTML: "transition-duration", React: "transitionDuration"},
	"TransitionProperty":          &typ{HTML: "transition-property", React: "transitionProperty"},
	"TransitionTimingFunction":    &typ{HTML: "transition-timing-function", React: "transitionTimingFunction"},
	"UnicodeBidi":                 &typ{HTML: "unicode-bidi", React: "unicodeBidi"},
	"UserSelect":                  &typ{HTML: "user-select", React: "userSelect"},
	"VerticalAlign":               &typ{HTML: "vertical-align", React: "verticalAlign"},
	"Visibility":                  &typ{HTML: "visibility", React: "visibility"},
	"VoiceBalance":                &typ{HTML: "voice-balance", React: "voiceBalance"},
	"VoiceDuration":               &typ{HTML: "voice-duration", React: "voiceDuration"},
	"VoiceFamily":                 &typ{HTML: "voice-family", React: "voiceFamily"},
	"VoicePitch":                  &typ{HTML: "voice-pitch", React: "voicePitch"},
	"VoiceRange":                  &typ{HTML: "voice-range", React: "voiceRange"},
	"VoiceRate":                   &typ{HTML: "voice-rate", React: "voiceRate"},
	"VoiceStress":                 &typ{HTML: "voice-stress", React: "voiceStress"},
	"VoiceVolume":                 &typ{HTML: "voice-volume", React: "voiceVolume"},
	"WhiteSpace":                  &typ{HTML: "white-space", React: "whiteSpace"},
	"Widows":                      &typ{HTML: "widows", React: "widows"},
	"Width":                       &typ{HTML: "width", React: "width"},
	"WillChange":                  &typ{HTML: "will-change", React: "willChange"},
	"WordBreak":                   &typ{HTML: "word-break", React: "wordBreak"},
	"WordSpacing":                 &typ{HTML: "word-spacing", React: "wordSpacing"},
	"WordWrap":                    &typ{HTML: "word-wrap", React: "wordWrap"},
	"WrapFlow":                    &typ{HTML: "wrap-flow", React: "wrapFlow"},
	"WrapThrough":                 &typ{HTML: "wrap-through", React: "wrapThrough"},
	"WritingMode":                 &typ{HTML: "writing-mode", React: "writingMode"},
	"ZIndex":                      &typ{HTML: "z-index", React: "zIndex"},
}

const (
	cssGenCmd = "cssGen"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix(cssGenCmd + ": ")

	flag.Parse()

	for n, a := range attrs {
		a.Name = n
		if a.React == "" {
			a.React = lowerInitial(n)
		}
		if a.HTML == "" {
			a.HTML = strings.ToLower(n)
		}
		if a.Type == "" {
			a.Type = "string"
		}
	}

	write := func(tmpl string, fn string) {
		buf := bytes.NewBuffer(nil)

		t, err := template.New("t").Parse(tmpl)
		if err != nil {
			fatalf("could not parse template: %v", err)
		}

		err = t.Execute(buf, attrs)
		if err != nil {
			fatalf("could not execute template: %v", err)
		}

		toWrite := buf.Bytes()
		out, err := format.Source(toWrite)
		if err == nil {
			toWrite = out
		}

		if err := ioutil.WriteFile(fn, toWrite, 0644); err != nil {
			fatalf("could not write %v: %v", fn, err)
		}
	}

	write(tmpl, gogenerate.NameFile("react", cssGenCmd))
	write(jsxTmpl, filepath.Join("jsx", gogenerate.NameFile("jsx", cssGenCmd)))
}

func lowerInitial(s string) string {
	if s == "" {
		return ""
	}

	r, w := utf8.DecodeRuneInString(s)
	return strings.ToLower(string(r)) + s[w:]
}

type typ struct {
	Name string

	// React is the React property name if not equivalent to the lower-initial
	// camel-case version of .Name
	React string

	// HTML is the HTML property name if not equivalent to the lowercase version
	// of .Name
	HTML string

	// Type is the type. Default is "string"
	Type string
}

var tmpl = `
 // Code generated by cssGen. DO NOT EDIT.

package react

import "github.com/gopherjs/gopherjs/js"

// CSS defines CSS attributes for HTML components. Largely based on
// https://developer.mozilla.org/en-US/docs/Web/CSS/Reference
//
type CSS struct {
	o *js.Object

	{{range . }}
	{{.Name}} {{.Type}}
	{{- end}}
}

// TODO: until we have a resolution on
// https://github.com/gopherjs/gopherjs/issues/236 we define hack() below

func (c *CSS) hack() *CSS {
	if c == nil {
		return nil
	}

	o := object.New()

	{{range . }}
	o.Set("{{.React}}", c.{{.Name}})
	{{- end}}

	return &CSS{o: o}
}
`

var jsxTmpl = `
package jsx

import (
	"fmt"
	"strings"

	"myitcv.io/react"
)

func parseCSS(s string) *react.CSS {
	res := new(react.CSS)

	parts := strings.Split(s, ";")

	for _, p := range parts {
		kv := strings.Split(p, ":")
		if len(kv) != 2 {
			panic(fmt.Errorf("invalid key-val %q in %q", p, s))
		}

		k, v := kv[0], kv[1]

		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, "\"")

		switch k {
		{{range .}}
		case "{{.HTML}}":
			res.{{.Name}} = v
		{{end}}
		default:
			panic(fmt.Errorf("unknown CSS key %q in %q", k, s))
		}
	}

	return res
}
`

func fatalf(format string, args ...interface{}) {
	panic(fmt.Errorf(format, args...))
}
