package app

import (
	"gart.dev/ui"
)

// This file holds the theme of the sample. The sample carries the same values
// in ui/core/themes/colors.dart, theme.dart, and dimens.dart.
//
// The sample reads the theme with Theme.of(context). A Go value holds no
// BuildContext, thus the program names the 2 themes with ui.UseTheme and it
// reads the active 1 with ui.Theme().

// The colors of the sample.
const (
	colorBlack1           = ui.Color(0xFF101010)
	colorWhite1           = ui.Color(0xFFFFF7FA)
	colorGrey1            = ui.Color(0xFFF2F2F2)
	colorGrey3            = ui.Color(0xFFA4A4A4)
	colorWhiteTransparent = ui.Color(0x4DFFFFFF)
	colorBlackTransparent = ui.Color(0x4D000000)
	colorRed1             = ui.Color(0xFFE74C3C)
	colorWhite            = ui.Color(0xFFFFFFFF)
	colorBlack            = ui.Color(0xFF000000)
	colorRed              = ui.Color(0xFFF44336)
	colorPurple700        = ui.Color(0xFF7B1FA2)
	colorPurple400        = ui.Color(0xFFAB47BC)
	colorGrey200          = ui.Color(0xFFEEEEEE)
)

// The 2 fonts of the sample. The sample reads each 1 from google_fonts, and
// this port ships the 2 files under assets/fonts.
const (
	fontRubik    = "Rubik"
	fontOpenSans = "OpenSans"
)

// weight gives the axis of a variable font. A face of such a font stands
// between 2 weights, thus the style names the axis and not a face.
func weight(v float64) []ui.FontVariation {
	return []ui.FontVariation{{Axis: "wght", Value: v}}
}

// textTheme holds the 8 styles that the sample names.
func textTheme() *ui.TextTheme {
	return &ui.TextTheme{
		HeadlineLarge: &ui.TextStyle{Size: ui.P(32.0), Weight: "w500", Variations: weight(500)},
		HeadlineSmall: &ui.TextStyle{Size: ui.P(18.0), Weight: "w400", Variations: weight(400)},
		TitleMedium:   &ui.TextStyle{Size: ui.P(18.0), Weight: "w500", Variations: weight(500)},
		BodyLarge:     &ui.TextStyle{Size: ui.P(18.0), Weight: "w400", Variations: weight(400)},
		BodyMedium:    &ui.TextStyle{Size: ui.P(16.0), Weight: "w400", Variations: weight(400)},
		BodySmall: &ui.TextStyle{
			Size: ui.P(14.0), Weight: "w400", Variations: weight(400),
			Color: ui.P(colorGrey3),
		},
		LabelSmall: &ui.TextStyle{
			Size: ui.P(10.0), Weight: "w500", Variations: weight(500),
			Color: ui.P(colorGrey3),
		},
		LabelLarge: &ui.TextStyle{
			Size: ui.P(18.0), Weight: "w400", Variations: weight(400),
			Color: ui.P(colorGrey3),
		},
	}
}

// hintStyle holds the text of a field that carries no value. The color works
// for the 2 brightnesses.
func hintStyle() *ui.TextStyle {
	return &ui.TextStyle{
		Color: ui.P(colorGrey3), Size: ui.P(18.0), Weight: "w400",
		Variations: weight(400),
	}
}

// lightTheme holds the theme of the light brightness.
var lightTheme = ui.ThemeStyle{
	Brightness: "light",
	ColorScheme: &ui.ColorScheme{
		Brightness:  "light",
		Primary:     colorBlack1,
		OnPrimary:   colorWhite1,
		Secondary:   colorBlack1,
		OnSecondary: colorWhite1,
		Surface:     colorWhite,
		OnSurface:   colorBlack1,
		Error:       colorWhite,
		OnError:     colorRed,
	},
	TextTheme:       textTheme(),
	InputDecoration: &ui.InputDecorationStyle{HintStyle: hintStyle()},
}

// darkTheme holds the theme of the dark brightness.
var darkTheme = ui.ThemeStyle{
	Brightness: "dark",
	ColorScheme: &ui.ColorScheme{
		Brightness:  "dark",
		Primary:     colorWhite1,
		OnPrimary:   colorBlack1,
		Secondary:   colorWhite1,
		OnSecondary: colorBlack1,
		Surface:     colorBlack1,
		OnSurface:   colorWhite,
		Error:       colorBlack,
		OnError:     colorRed1,
	},
	TextTheme:       textTheme(),
	InputDecoration: &ui.InputDecorationStyle{HintStyle: hintStyle()},
}

// chipColors gives the 2 colors of a tag chip. The sample holds them in a
// ThemeExtension, and a Go program holds them in a function of its own.
func chipColors() (chip, onChip ui.Color) {
	if ui.Theme().Dark {
		return colorBlackTransparent, colorWhite
	}
	return colorWhiteTransparent, colorWhite
}

// The padding of the sample. Dimens of the sample holds the same 2 values.
const (
	paddingHorizontal = 20.0
	paddingVertical   = 24.0
)

// dimens holds the padding of the screen and the size of the picture of the
// account. The sample reads MediaQuery.sizeOf(context), and this port reads
// ui.Media().
type dimens struct {
	ScreenHorizontal float64
	ScreenVertical   float64
	ProfilePicture   float64
}

// screen gives the padding of the width of the screen. The sample gives the
// desktop values between 600 and 840 alone.
func screen() dimens {
	w := ui.Media().Width
	if w > 600 && w < 840 {
		return dimens{ScreenHorizontal: 100, ScreenVertical: 64, ProfilePicture: 128}
	}
	return dimens{ScreenHorizontal: 20, ScreenVertical: 24, ProfilePicture: 64}
}

// screenSides gives the padding of the 4 edges of a screen.
func screenSides() ui.Insets {
	d := screen()
	return ui.Sides(d.ScreenHorizontal, d.ScreenVertical)
}

// screenX gives the horizontal padding of a screen.
func screenX() ui.Insets {
	d := screen()
	return ui.Sides(d.ScreenHorizontal, 0)
}
