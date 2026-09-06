// nodefilter:values

package ui

// This file holds the value type of each theme of 1 widget family, for example
// SliderThemeData. Each 1 carries the subset of the fields of the Flutter
// theme that a Go value holds. The host of flutter/goflutter builds the real
// theme from it.
//
// A field that Flutter carries for each state holds 1 value here, as
// ui.ButtonStyle does. A text field names a value of an enum of Flutter, and
// an empty text gives the default of Flutter.
//
// A theme that holds 1 ButtonStyle alone, for example ElevatedButtonThemeData,
// reads ui.ButtonStyle and thus takes no type of its own.

// ActionIconStyle holds the part of the Flutter ActionIconThemeData that a
// node field reads.
//
// Each builder gives the icon of 1 button of an app bar.
type ActionIconStyle struct {
	BackButtonIconBuilder      Builder
	CloseButtonIconBuilder     Builder
	DrawerButtonIconBuilder    Builder
	EndDrawerButtonIconBuilder Builder
}

// AppBarStyle holds the part of the Flutter AppBarThemeData that a node field
// reads.
//
// systemOverlayStyle stays out, because no Go value holds a
// SystemUiOverlayStyle. The deprecated color stays out, because
// BackgroundColor carries the same value.
type AppBarStyle struct {
	BackgroundColor        *Color
	ForegroundColor        *Color
	Elevation              *float64
	ScrolledUnderElevation *float64
	ShadowColor            *Color
	SurfaceTintColor       *Color
	Shape                  *Shape
	IconTheme              *IconStyle
	ActionsIconTheme       *IconStyle
	CenterTitle            *bool
	TitleSpacing           *float64
	LeadingWidth           *float64
	ToolbarHeight          *float64
	ToolbarTextStyle       *TextStyle
	TitleTextStyle         *TextStyle
	ActionsPadding         *Insets
}

// BadgeStyle holds the part of the Flutter BadgeThemeData that a node field
// reads.
type BadgeStyle struct {
	BackgroundColor *Color
	TextColor       *Color
	SmallSize       *float64
	LargeSize       *float64
	TextStyle       *TextStyle
	Padding         *Insets
	Alignment       string
	Offset          *Offset
}

// BottomAppBarStyle holds the part of the Flutter BottomAppBarThemeData that a
// node field reads.
//
// shape stays out, because a Dart subclass must implement a NotchedShape.
type BottomAppBarStyle struct {
	Color            *Color
	Elevation        *float64
	Height           *float64
	SurfaceTintColor *Color
	ShadowColor      *Color
	Padding          *Insets
}

// BottomNavigationBarStyle holds the part of the Flutter
// BottomNavigationBarThemeData that a node field reads.
type BottomNavigationBarStyle struct {
	BackgroundColor      *Color
	Elevation            *float64
	SelectedIconTheme    *IconStyle
	UnselectedIconTheme  *IconStyle
	SelectedItemColor    *Color
	UnselectedItemColor  *Color
	SelectedLabelStyle   *TextStyle
	UnselectedLabelStyle *TextStyle
	ShowSelectedLabels   *bool
	ShowUnselectedLabels *bool
	Type                 string
	EnableFeedback       *bool
	LandscapeLayout      string
	MouseCursor          string
}

// ButtonBarStyle holds the part of the Flutter ButtonBarThemeData that a node
// field reads.
type ButtonBarStyle struct {
	Alignment             string
	MainAxisSize          string
	ButtonTextTheme       string
	ButtonMinWidth        *float64
	ButtonHeight          *float64
	ButtonPadding         *Insets
	ButtonAlignedDropdown *bool
	LayoutBehavior        string
	OverflowDirection     string
}

// ButtonThemeStyle holds the part of the Flutter ButtonThemeData that a node
// field reads.
//
// The name differs from the Flutter name, because ui.ButtonStyle already
// holds the Flutter ButtonStyle.
type ButtonThemeStyle struct {
	TextTheme             string
	MinWidth              *float64
	Height                *float64
	Padding               *Insets
	Shape                 *Shape
	LayoutBehavior        string
	AlignedDropdown       *bool
	ButtonColor           *Color
	DisabledColor         *Color
	FocusColor            *Color
	HoverColor            *Color
	HighlightColor        *Color
	SplashColor           *Color
	MaterialTapTargetSize string
}

// CardStyle holds the part of the Flutter CardThemeData that a node field
// reads.
//
// ClipBehavior names a value of the Flutter enum Clip.
type CardStyle struct {
	ClipBehavior     string
	Color            *Color
	ShadowColor      *Color
	SurfaceTintColor *Color
	Elevation        *float64
	Margin           *Insets
	Shape            *Shape
}

// CarouselViewStyle holds the part of the Flutter CarouselViewThemeData that a
// node field reads.
type CarouselViewStyle struct {
	Elevation        *float64
	BackgroundColor  *Color
	OverlayColor     *StateColor
	Shape            *Shape
	Padding          *Insets
	ItemClipBehavior string
}

// CheckboxStyle holds the part of the Flutter CheckboxThemeData that a node
// field reads.
type CheckboxStyle struct {
	MouseCursor           string
	FillColor             *StateColor
	CheckColor            *StateColor
	OverlayColor          *StateColor
	SplashRadius          *float64
	MaterialTapTargetSize string
	VisualDensity         *Density
	Shape                 *Shape
	Side                  *BorderSide
}

// ChipStyle holds the part of the Flutter ChipThemeData that a node field
// reads.
type ChipStyle struct {
	Color                    *StateColor
	BackgroundColor          *Color
	DeleteIconColor          *Color
	DisabledColor            *Color
	SelectedColor            *Color
	SecondarySelectedColor   *Color
	ShadowColor              *Color
	SurfaceTintColor         *Color
	SelectedShadowColor      *Color
	ShowCheckmark            *bool
	CheckmarkColor           *Color
	LabelPadding             *Insets
	Padding                  *Insets
	Side                     *BorderSide
	Shape                    *Shape
	LabelStyle               *TextStyle
	SecondaryLabelStyle      *TextStyle
	Brightness               string
	Elevation                *float64
	PressElevation           *float64
	IconTheme                *IconStyle
	AvatarBoxConstraints     *Constraints
	DeleteIconBoxConstraints *Constraints
}

// CupertinoStyle holds the part of the Flutter CupertinoThemeData that a node
// field reads.
type CupertinoStyle struct {
	Brightness              string
	PrimaryColor            *Color
	PrimaryContrastingColor *Color
	BarBackgroundColor      *Color
	ScaffoldBackgroundColor *Color
	SelectionHandleColor    *Color
	ApplyThemeToAll         *bool
}

// DataTableStyle holds the part of the Flutter DataTableThemeData that a node
// field reads.
type DataTableStyle struct {
	Decoration               *Decoration
	DataRowColor             *StateColor
	DataRowMinHeight         *float64
	DataRowMaxHeight         *float64
	DataTextStyle            *TextStyle
	HeadingRowColor          *StateColor
	HeadingRowHeight         *float64
	HeadingTextStyle         *TextStyle
	HorizontalMargin         *float64
	ColumnSpacing            *float64
	DividerThickness         *float64
	CheckboxHorizontalMargin *float64
	HeadingCellCursor        string
	DataRowCursor            string
	HeadingRowAlignment      string
}

// DatePickerStyle holds the part of the Flutter DatePickerThemeData that a
// node field reads.
type DatePickerStyle struct {
	BackgroundColor                  *Color
	Elevation                        *float64
	ShadowColor                      *Color
	SurfaceTintColor                 *Color
	Shape                            *Shape
	HeaderBackgroundColor            *Color
	HeaderForegroundColor            *Color
	HeaderHeadlineStyle              *TextStyle
	HeaderHelpStyle                  *TextStyle
	WeekdayStyle                     *TextStyle
	DayStyle                         *TextStyle
	DayForegroundColor               *StateColor
	DayBackgroundColor               *StateColor
	DayOverlayColor                  *StateColor
	DayShape                         *Shape
	TodayForegroundColor             *StateColor
	TodayBackgroundColor             *StateColor
	TodayBorder                      *BorderSide
	YearStyle                        *TextStyle
	YearForegroundColor              *StateColor
	YearBackgroundColor              *StateColor
	YearOverlayColor                 *StateColor
	YearShape                        *Shape
	RangePickerBackgroundColor       *Color
	RangePickerElevation             *float64
	RangePickerShadowColor           *Color
	RangePickerSurfaceTintColor      *Color
	RangePickerShape                 *Shape
	RangePickerHeaderBackgroundColor *Color
	RangePickerHeaderForegroundColor *Color
	RangePickerHeaderHeadlineStyle   *TextStyle
	RangePickerHeaderHelpStyle       *TextStyle
	RangeSelectionBackgroundColor    *Color
	RangeSelectionOverlayColor       *StateColor
	DividerColor                     *Color
	CancelButtonStyle                *ButtonStyle
	ConfirmButtonStyle               *ButtonStyle
	Locale                           string
	ToggleButtonTextStyle            *TextStyle
	SubHeaderForegroundColor         *Color
}

// DialogStyle holds the part of the Flutter DialogThemeData that a node field
// reads.
//
// Alignment names a value of the alignment table of the host. ClipBehavior
// names a value of the Flutter enum Clip.
type DialogStyle struct {
	BackgroundColor  *Color
	Elevation        *float64
	ShadowColor      *Color
	SurfaceTintColor *Color
	Shape            *Shape
	Alignment        string
	IconColor        *Color
	TitleTextStyle   *TextStyle
	ContentTextStyle *TextStyle
	ActionsPadding   *Insets
	BarrierColor     *Color
	InsetPadding     *Insets
	ClipBehavior     string
	Constraints      *Constraints
}

// DividerStyle holds the part of the Flutter DividerThemeData that a node
// field reads.
type DividerStyle struct {
	Color     *Color
	Space     *float64
	Thickness *float64
	Indent    *float64
	EndIndent *float64
	Radius    *Radius
}

// DrawerStyle holds the part of the Flutter DrawerThemeData that a node field
// reads.
type DrawerStyle struct {
	BackgroundColor  *Color
	ScrimColor       *Color
	Elevation        *float64
	ShadowColor      *Color
	SurfaceTintColor *Color
	Shape            *Shape
	EndShape         *Shape
	Width            *float64
	ClipBehavior     string
}

// DropdownMenuStyle holds the part of the Flutter DropdownMenuThemeData that a
// node field reads.
//
// MenuStyle gives the style of the menu that the button opens.
type DropdownMenuStyle struct {
	TextStyle     *TextStyle
	MenuStyle     *MenuStyle
	DisabledColor *Color
}

// ExpansionTileStyle holds the part of the Flutter ExpansionTileThemeData that
// a node field reads.
type ExpansionTileStyle struct {
	BackgroundColor          *Color
	CollapsedBackgroundColor *Color
	TilePadding              *Insets
	ExpandedAlignment        string
	ChildrenPadding          *Insets
	IconColor                *Color
	CollapsedIconColor       *Color
	TextColor                *Color
	CollapsedTextColor       *Color
	Shape                    *Shape
	CollapsedShape           *Shape
	ClipBehavior             string
}

// FloatingActionButtonStyle holds the part of the Flutter
// FloatingActionButtonThemeData that a node field reads.
type FloatingActionButtonStyle struct {
	ForegroundColor          *Color
	BackgroundColor          *Color
	FocusColor               *Color
	HoverColor               *Color
	SplashColor              *Color
	Elevation                *float64
	FocusElevation           *float64
	HoverElevation           *float64
	DisabledElevation        *float64
	HighlightElevation       *float64
	Shape                    *Shape
	EnableFeedback           *bool
	IconSize                 *float64
	SizeConstraints          *Constraints
	SmallSizeConstraints     *Constraints
	LargeSizeConstraints     *Constraints
	ExtendedSizeConstraints  *Constraints
	ExtendedIconLabelSpacing *float64
	ExtendedPadding          *Insets
	ExtendedTextStyle        *TextStyle
	MouseCursor              string
}

// InputDecorationStyle holds the part of the Flutter InputDecorationThemeData
// that a node field reads.
//
// FloatingLabelBehavior takes "never", "auto", or "always". HintFadeDuration
// holds a count of milliseconds.
//
// The 6 border fields stay out, because a Dart subclass must implement an
// InputBorder and ui.Shape gives an OutlinedBorder. floatingLabelAlignment
// stays out, because a FloatingLabelAlignment is a Dart object and no Go value
// holds 1.
type InputDecorationStyle struct {
	LabelStyle            *TextStyle
	FloatingLabelStyle    *TextStyle
	HelperStyle           *TextStyle
	HelperMaxLines        *int
	HintStyle             *TextStyle
	HintFadeDuration      *int
	HintMaxLines          *int
	ErrorStyle            *TextStyle
	ErrorMaxLines         *int
	FloatingLabelBehavior string
	IsDense               *bool
	ContentPadding        *Insets
	IsCollapsed           *bool
	IconColor             *Color
	PrefixStyle           *TextStyle
	PrefixIconColor       *Color
	PrefixIconConstraints *Constraints
	SuffixStyle           *TextStyle
	SuffixIconColor       *Color
	SuffixIconConstraints *Constraints
	CounterStyle          *TextStyle
	Filled                *bool
	FillColor             *Color
	ActiveIndicatorBorder *BorderSide
	OutlineBorder         *BorderSide
	FocusColor            *Color
	HoverColor            *Color
	AlignLabelWithHint    *bool
	Constraints           *Constraints
	VisualDensity         *Density
}

// ListTileStyle holds the part of the Flutter ListTileThemeData that a node
// field reads.
//
// Style names a value of the Flutter enum ListTileStyle, thus it takes "list"
// or "drawer". TitleAlignment names a value of ListTileTitleAlignment, and
// ControlAffinity names a value of ListTileControlAffinity. MouseCursor names
// a value of the cursor table of the host, and the host gives that value to
// each state.
type ListTileStyle struct {
	Dense                       *bool
	Shape                       *Shape
	Style                       string
	SelectedColor               *Color
	IconColor                   *Color
	TextColor                   *Color
	TitleTextStyle              *TextStyle
	SubtitleTextStyle           *TextStyle
	LeadingAndTrailingTextStyle *TextStyle
	ContentPadding              *Insets
	TileColor                   *Color
	SelectedTileColor           *Color
	HorizontalTitleGap          *float64
	MinVerticalPadding          *float64
	MinLeadingWidth             *float64
	MinTileHeight               *float64
	EnableFeedback              *bool
	MouseCursor                 string
	VisualDensity               *Density
	TitleAlignment              string
	ControlAffinity             string
	IsThreeLine                 *bool
}

// MaterialBannerStyle holds the part of the Flutter MaterialBannerThemeData
// that a node field reads.
type MaterialBannerStyle struct {
	BackgroundColor  *Color
	SurfaceTintColor *Color
	ShadowColor      *Color
	DividerColor     *Color
	ContentTextStyle *TextStyle
	Elevation        *float64
	Padding          *Insets
	LeadingPadding   *Insets
}

// MenuStyle holds the part of the Flutter MenuStyle that a node field reads.
//
// MenuThemeData and MenuBarThemeData each hold 1 MenuStyle alone, thus a
// field of that type reads this struct too.
type MenuStyle struct {
	BackgroundColor  *StateColor
	ShadowColor      *StateColor
	SurfaceTintColor *StateColor
	Elevation        *float64
	Padding          *Insets
	MinimumSize      *Size
	FixedSize        *Size
	MaximumSize      *Size
	Side             *BorderSide
	Shape            *Shape
	MouseCursor      string
	VisualDensity    *Density
	Alignment        string
}

// NavigationBarStyle holds the part of the Flutter NavigationBarThemeData that
// a node field reads.
type NavigationBarStyle struct {
	Height           *float64
	BackgroundColor  *Color
	Elevation        *float64
	ShadowColor      *Color
	SurfaceTintColor *Color
	IndicatorColor   *Color
	IndicatorShape   *Shape
	LabelTextStyle   *TextStyle
	IconTheme        *IconStyle
	LabelBehavior    string
	OverlayColor     *StateColor
	LabelPadding     *Insets
}

// NavigationDrawerStyle holds the part of the Flutter
// NavigationDrawerThemeData that a node field reads.
type NavigationDrawerStyle struct {
	TileHeight       *float64
	BackgroundColor  *Color
	Elevation        *float64
	ShadowColor      *Color
	SurfaceTintColor *Color
	IndicatorColor   *Color
	IndicatorShape   *Shape
	IndicatorSize    *Size
	LabelTextStyle   *TextStyle
	IconTheme        *IconStyle
}

// NavigationRailStyle holds the part of the Flutter NavigationRailThemeData
// that a node field reads.
type NavigationRailStyle struct {
	BackgroundColor          *Color
	Elevation                *float64
	UnselectedLabelTextStyle *TextStyle
	SelectedLabelTextStyle   *TextStyle
	UnselectedIconTheme      *IconStyle
	SelectedIconTheme        *IconStyle
	GroupAlignment           *float64
	LabelType                string
	UseIndicator             *bool
	IndicatorColor           *Color
	IndicatorShape           *Shape
	MinWidth                 *float64
	MinExtendedWidth         *float64
}

// PopupMenuStyle holds the part of the Flutter PopupMenuThemeData that a node
// field reads.
type PopupMenuStyle struct {
	Color            *Color
	Shape            *Shape
	MenuPadding      *Insets
	Elevation        *float64
	ShadowColor      *Color
	SurfaceTintColor *Color
	TextStyle        *TextStyle
	LabelTextStyle   *TextStyle
	EnableFeedback   *bool
	MouseCursor      string
	Position         string
	IconColor        *Color
	IconSize         *float64
}

// ProgressIndicatorStyle holds the part of the Flutter
// ProgressIndicatorThemeData that a node field reads.
type ProgressIndicatorStyle struct {
	Color                  *Color
	LinearTrackColor       *Color
	LinearMinHeight        *float64
	CircularTrackColor     *Color
	RefreshBackgroundColor *Color
	BorderRadius           *Radius
	StopIndicatorColor     *Color
	StopIndicatorRadius    *float64
	StrokeWidth            *float64
	StrokeAlign            *float64
	StrokeCap              string
	Constraints            *Constraints
	TrackGap               *float64
	CircularTrackPadding   *Insets
}

// RadioStyle holds the part of the Flutter RadioThemeData that a node field
// reads.
type RadioStyle struct {
	MouseCursor           string
	FillColor             *StateColor
	OverlayColor          *StateColor
	SplashRadius          *float64
	MaterialTapTargetSize string
	VisualDensity         *Density
	BackgroundColor       *StateColor
	Side                  *BorderSide
	InnerRadius           *float64
}

// ScrollbarStyle holds the part of the Flutter ScrollbarThemeData that a node
// field reads.
type ScrollbarStyle struct {
	ThumbVisibility *bool
	Thickness       *float64
	TrackVisibility *bool

	// Radius gives a circular radius of the corners of the thumb. An
	// elliptical radius stays out.
	Radius *float64

	ThumbColor       *StateColor
	TrackColor       *StateColor
	TrackBorderColor *StateColor
	CrossAxisMargin  *float64
	MainAxisMargin   *float64
	MinThumbLength   *float64
	Interactive      *bool
}

// SearchBarStyle holds the part of the Flutter SearchBarThemeData that a node
// field reads.
type SearchBarStyle struct {
	Elevation          *float64
	BackgroundColor    *StateColor
	ShadowColor        *StateColor
	SurfaceTintColor   *StateColor
	OverlayColor       *StateColor
	Side               *BorderSide
	Shape              *Shape
	Padding            *Insets
	TextStyle          *TextStyle
	HintStyle          *TextStyle
	Constraints        *Constraints
	TextCapitalization string
}

// SearchViewStyle holds the part of the Flutter SearchViewThemeData that a
// node field reads.
type SearchViewStyle struct {
	BackgroundColor  *Color
	Elevation        *float64
	SurfaceTintColor *Color
	Constraints      *Constraints
	Padding          *Insets
	BarPadding       *Insets
	ShrinkWrap       *bool
	Side             *BorderSide
	Shape            *Shape
	HeaderHeight     *float64
	HeaderTextStyle  *TextStyle
	HeaderHintStyle  *TextStyle
	DividerColor     *Color
}

// SegmentedButtonStyle holds the part of the Flutter SegmentedButtonThemeData
// that a node field reads.
type SegmentedButtonStyle struct {
	Style        *ButtonStyle
	SelectedIcon Node
}

// SliderStyle holds the part of the Flutter SliderThemeData that a node field
// reads.
type SliderStyle struct {
	TrackHeight                       *float64
	ActiveTrackColor                  *Color
	InactiveTrackColor                *Color
	SecondaryActiveTrackColor         *Color
	DisabledActiveTrackColor          *Color
	DisabledInactiveTrackColor        *Color
	DisabledSecondaryActiveTrackColor *Color
	ActiveTickMarkColor               *Color
	InactiveTickMarkColor             *Color
	DisabledActiveTickMarkColor       *Color
	DisabledInactiveTickMarkColor     *Color
	ThumbColor                        *Color
	OverlappingShapeStrokeColor       *Color
	DisabledThumbColor                *Color
	OverlayColor                      *Color
	ValueIndicatorColor               *Color
	ValueIndicatorStrokeColor         *Color
	ShowValueIndicator                string
	ValueIndicatorTextStyle           *TextStyle
	MinThumbSeparation                *float64
	MouseCursor                       string
	AllowedInteraction                string
	Padding                           *Insets
	ThumbSize                         *Size
	TrackGap                          *float64
}

// SnackBarStyle holds the part of the Flutter SnackBarThemeData that a node
// field reads.
type SnackBarStyle struct {
	BackgroundColor               *Color
	ActionTextColor               *Color
	DisabledActionTextColor       *Color
	ContentTextStyle              *TextStyle
	Elevation                     *float64
	Shape                         *Shape
	Behavior                      string
	Width                         *float64
	InsetPadding                  *Insets
	ShowCloseIcon                 *bool
	CloseIconColor                *Color
	ActionOverflowThreshold       *float64
	ActionBackgroundColor         *Color
	DisabledActionBackgroundColor *Color
	DismissDirection              string
}

// SwitchStyle holds the part of the Flutter SwitchThemeData that a node field
// reads.
type SwitchStyle struct {
	ThumbColor            *StateColor
	TrackColor            *StateColor
	TrackOutlineColor     *StateColor
	TrackOutlineWidth     *float64
	MaterialTapTargetSize string
	MouseCursor           string
	OverlayColor          *StateColor
	SplashRadius          *float64
	Padding               *Insets
}

// TabBarStyle holds the part of the Flutter TabBarThemeData that a node field
// reads.
//
// IndicatorSize takes "tab" or "label". TabAlignment and IndicatorAnimation
// each name a value of the Flutter enum of the same name. MouseCursor names a
// value of the cursor table of the host, and the host gives that value to each
// state.
//
// splashFactory stays out, because a Dart subclass must implement an
// InteractiveInkFeatureFactory. textScaler stays out, because no Go value
// holds a TextScaler.
type TabBarStyle struct {
	Indicator            *Decoration
	IndicatorColor       *Color
	IndicatorSize        string
	DividerColor         *Color
	DividerHeight        *float64
	LabelColor           *Color
	LabelPadding         *Insets
	LabelStyle           *TextStyle
	UnselectedLabelColor *Color
	UnselectedLabelStyle *TextStyle
	OverlayColor         *StateColor
	MouseCursor          string
	TabAlignment         string
	IndicatorAnimation   string
	SplashBorderRadius   *Radius
}

// TextSelectionStyle holds the part of the Flutter TextSelectionThemeData that
// a node field reads.
type TextSelectionStyle struct {
	CursorColor          *Color
	SelectionColor       *Color
	SelectionHandleColor *Color
}

// TimePickerStyle holds the part of the Flutter TimePickerThemeData that a
// node field reads.
type TimePickerStyle struct {
	BackgroundColor                *Color
	CancelButtonStyle              *ButtonStyle
	ConfirmButtonStyle             *ButtonStyle
	DayPeriodBorderSide            *BorderSide
	DayPeriodColor                 *Color
	DayPeriodShape                 *Shape
	DayPeriodTextColor             *Color
	DayPeriodTextStyle             *TextStyle
	DialBackgroundColor            *Color
	DialHandColor                  *Color
	DialTextColor                  *Color
	DialTextStyle                  *TextStyle
	Elevation                      *float64
	EntryModeIconColor             *Color
	HelpTextStyle                  *TextStyle
	HourMinuteColor                *Color
	HourMinuteShape                *Shape
	HourMinuteTextColor            *Color
	HourMinuteTextStyle            *TextStyle
	Padding                        *Insets
	Shape                          *Shape
	TimeSelectorSeparatorColor     *StateColor
	TimeSelectorSeparatorTextStyle *TextStyle
}

// ToggleButtonsStyle holds the part of the Flutter ToggleButtonsThemeData that
// a node field reads.
type ToggleButtonsStyle struct {
	TextStyle           *TextStyle
	Constraints         *Constraints
	Color               *Color
	SelectedColor       *Color
	DisabledColor       *Color
	FillColor           *Color
	FocusColor          *Color
	HighlightColor      *Color
	HoverColor          *Color
	SplashColor         *Color
	BorderColor         *Color
	SelectedBorderColor *Color
	DisabledBorderColor *Color
	BorderRadius        *Radius
	BorderWidth         *float64
}

// TooltipStyle holds the part of the Flutter TooltipThemeData that a node
// field reads.
type TooltipStyle struct {
	Constraints          *Constraints
	Padding              *Insets
	Margin               *Insets
	VerticalOffset       *float64
	PreferBelow          *bool
	ExcludeFromSemantics *bool
	Decoration           *Decoration
	TextStyle            *TextStyle
	TextAlign            string
	WaitDuration         *int
	ShowDuration         *int
	ExitDuration         *int
	TriggerMode          string
	EnableFeedback       *bool
}
