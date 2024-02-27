package ui

// Color representation
type Color uint32

// Colors
const (
	Grey     Color = Color233
	Grey2    Color = Color244
	Darkgrey Color = Color232
)

// First 256 entries correspond to Colors from the color palette as defined by the standard
// https://en.wikipedia.org/wiki/ANSI_escape_code#Colors. Order is important.
// Below that, a few colors from the TrueColor palette are defined.
const (
	ColorBlack Color = iota
	ColorMaroon
	ColorGreen
	ColorOlive
	ColorNavy
	ColorPurple
	ColorTeal
	ColorSilver
	ColorGray
	ColorRed
	ColorLime
	ColorYellow
	ColorBlue
	ColorFuchsia
	ColorAqua
	ColorWhite
	Color16
	Color17
	Color18
	Color19
	Color20
	Color21
	Color22
	Color23
	Color24
	Color25
	Color26
	Color27
	Color28
	Color29
	Color30
	Color31
	Color32
	Color33
	Color34
	Color35
	Color36
	Color37
	Color38
	Color39
	Color40
	Color41
	Color42
	Color43
	Color44
	Color45
	Color46
	Color47
	Color48
	Color49
	Color50
	Color51
	Color52
	Color53
	Color54
	Color55
	Color56
	Color57
	Color58
	Color59
	Color60
	Color61
	Color62
	Color63
	Color64
	Color65
	Color66
	Color67
	Color68
	Color69
	Color70
	Color71
	Color72
	Color73
	Color74
	Color75
	Color76
	Color77
	Color78
	Color79
	Color80
	Color81
	Color82
	Color83
	Color84
	Color85
	Color86
	Color87
	Color88
	Color89
	Color90
	Color91
	Color92
	Color93
	Color94
	Color95
	Color96
	Color97
	Color98
	Color99
	Color100
	Color101
	Color102
	Color103
	Color104
	Color105
	Color106
	Color107
	Color108
	Color109
	Color110
	Color111
	Color112
	Color113
	Color114
	Color115
	Color116
	Color117
	Color118
	Color119
	Color120
	Color121
	Color122
	Color123
	Color124
	Color125
	Color126
	Color127
	Color128
	Color129
	Color130
	Color131
	Color132
	Color133
	Color134
	Color135
	Color136
	Color137
	Color138
	Color139
	Color140
	Color141
	Color142
	Color143
	Color144
	Color145
	Color146
	Color147
	Color148
	Color149
	Color150
	Color151
	Color152
	Color153
	Color154
	Color155
	Color156
	Color157
	Color158
	Color159
	Color160
	Color161
	Color162
	Color163
	Color164
	Color165
	Color166
	Color167
	Color168
	Color169
	Color170
	Color171
	Color172
	Color173
	Color174
	Color175
	Color176
	Color177
	Color178
	Color179
	Color180
	Color181
	Color182
	Color183
	Color184
	Color185
	Color186
	Color187
	Color188
	Color189
	Color190
	Color191
	Color192
	Color193
	Color194
	Color195
	Color196
	Color197
	Color198
	Color199
	Color200
	Color201
	Color202
	Color203
	Color204
	Color205
	Color206
	Color207
	Color208
	Color209
	Color210
	Color211
	Color212
	Color213
	Color214
	Color215
	Color216
	Color217
	Color218
	Color219
	Color220
	Color221
	Color222
	Color223
	Color224
	Color225
	Color226
	Color227
	Color228
	Color229
	Color230
	Color231
	Color232
	Color233
	Color234
	Color235
	Color236
	Color237
	Color238
	Color239
	Color240
	Color241
	Color242
	Color243
	Color244
	Color245
	Color246
	Color247
	Color248
	Color249
	Color250
	Color251
	Color252
	Color253
	Color254
	Color255
	ColorAliceBlue
	ColorAntiqueWhite
	ColorAquaMarine
	ColorAzure
	ColorBeige
	ColorBisque
	ColorBlanchedAlmond
	ColorBlueViolet
	ColorBrown
	ColorBurlyWood
	ColorCadetBlue
	ColorChartreuse
	ColorChocolate
	ColorCoral
	ColorCornflowerBlue
	ColorCornsilk
	ColorCrimson
	ColorDarkBlue
	ColorDarkCyan
	ColorDarkGoldenrod
	ColorDarkGray
	ColorDarkGreen
	ColorDarkKhaki
	ColorDarkMagenta
	ColorDarkOliveGreen
	ColorDarkOrange
	ColorDarkOrchid
	ColorDarkRed
	ColorDarkSalmon
	ColorDarkSeaGreen
	ColorDarkSlateBlue
	ColorDarkSlateGray
	ColorDarkTurquoise
	ColorDarkViolet
	ColorDeepPink
	ColorDeepSkyBlue
	ColorDimGray
	ColorDodgerBlue
	ColorFireBrick
	ColorFloralWhite
	ColorForestGreen
	ColorGainsboro
	ColorGhostWhite
	ColorGold
	ColorGoldenrod
	ColorGreenYellow
	ColorHoneydew
	ColorHotPink
	ColorIndianRed
	ColorIndigo
	ColorIvory
	ColorKhaki
	ColorLavender
	ColorLavenderBlush
	ColorLawnGreen
	ColorLemonChiffon
	ColorLightBlue
	ColorLightCoral
	ColorLightCyan
	ColorLightGoldenrodYellow
	ColorLightGray
	ColorLightGreen
	ColorLightPink
	ColorLightSalmon
	ColorLightSeaGreen
	ColorLightSkyBlue
	ColorLightSlateGray
	ColorLightSteelBlue
	ColorLightYellow
	ColorLimeGreen
	ColorLinen
	ColorMediumAquamarine
	ColorMediumBlue
	ColorMediumOrchid
	ColorMediumPurple
	ColorMediumSeaGreen
	ColorMediumSlateBlue
	ColorMediumSpringGreen
	ColorMediumTurquoise
	ColorMediumVioletRed
	ColorMidnightBlue
	ColorMintCream
	ColorMistyRose
	ColorMoccasin
	ColorNavajoWhite
	ColorOldLace
	ColorOliveDrab
	ColorOrange
	ColorOrangeRed
	ColorOrchid
	ColorPaleGoldenrod
	ColorPaleGreen
	ColorPaleTurquoise
	ColorPaleVioletRed
	ColorPapayaWhip
	ColorPeachPuff
	ColorPeru
	ColorPink
	ColorPlum
	ColorPowderBlue
	ColorRebeccaPurple
	ColorRosyBrown
	ColorRoyalBlue
	ColorSaddleBrown
	ColorSalmon
	ColorSandyBrown
	ColorSeaGreen
	ColorSeashell
	ColorSienna
	ColorSkyblue
	ColorSlateBlue
	ColorSlateGray
	ColorSnow
	ColorSpringGreen
	ColorSteelBlue
	ColorTan
	ColorThistle
	ColorTomato
	ColorTurquoise
	ColorViolet
	ColorWheat
	ColorWhiteSmoke
	ColorYellowGreen
)

var colorNames = map[string]Color{
	"black":                ColorBlack,
	"maroon":               ColorMaroon,
	"green":                ColorGreen,
	"olive":                ColorOlive,
	"navy":                 ColorNavy,
	"purple":               ColorPurple,
	"teal":                 ColorTeal,
	"silver":               ColorSilver,
	"gray":                 ColorGray,
	"red":                  ColorRed,
	"lime":                 ColorLime,
	"yellow":               ColorYellow,
	"blue":                 ColorBlue,
	"fuchsia":              ColorFuchsia,
	"aqua":                 ColorAqua,
	"white":                ColorWhite,
	"aliceblue":            ColorAliceBlue,
	"antiquewhite":         ColorAntiqueWhite,
	"aquamarine":           ColorAquaMarine,
	"azure":                ColorAzure,
	"beige":                ColorBeige,
	"bisque":               ColorBisque,
	"blanchedalmond":       ColorBlanchedAlmond,
	"blueviolet":           ColorBlueViolet,
	"brown":                ColorBrown,
	"burlywood":            ColorBurlyWood,
	"cadetblue":            ColorCadetBlue,
	"chartreuse":           ColorChartreuse,
	"chocolate":            ColorChocolate,
	"coral":                ColorCoral,
	"cornflowerblue":       ColorCornflowerBlue,
	"cornsilk":             ColorCornsilk,
	"crimson":              ColorCrimson,
	"darkblue":             ColorDarkBlue,
	"darkcyan":             ColorDarkCyan,
	"darkgoldenrod":        ColorDarkGoldenrod,
	"darkgray":             ColorDarkGray,
	"darkgreen":            ColorDarkGreen,
	"darkkhaki":            ColorDarkKhaki,
	"darkmagenta":          ColorDarkMagenta,
	"darkolivegreen":       ColorDarkOliveGreen,
	"darkorange":           ColorDarkOrange,
	"darkorchid":           ColorDarkOrchid,
	"darkred":              ColorDarkRed,
	"darksalmon":           ColorDarkSalmon,
	"darkseagreen":         ColorDarkSeaGreen,
	"darkslateblue":        ColorDarkSlateBlue,
	"darkslategray":        ColorDarkSlateGray,
	"darkturquoise":        ColorDarkTurquoise,
	"darkviolet":           ColorDarkViolet,
	"deeppink":             ColorDeepPink,
	"deepskyblue":          ColorDeepSkyBlue,
	"dimgray":              ColorDimGray,
	"dodgerblue":           ColorDodgerBlue,
	"firebrick":            ColorFireBrick,
	"floralwhite":          ColorFloralWhite,
	"forestgreen":          ColorForestGreen,
	"gainsboro":            ColorGainsboro,
	"ghostwhite":           ColorGhostWhite,
	"gold":                 ColorGold,
	"goldenrod":            ColorGoldenrod,
	"greenyellow":          ColorGreenYellow,
	"honeydew":             ColorHoneydew,
	"hotpink":              ColorHotPink,
	"indianred":            ColorIndianRed,
	"indigo":               ColorIndigo,
	"ivory":                ColorIvory,
	"khaki":                ColorKhaki,
	"lavender":             ColorLavender,
	"lavenderblush":        ColorLavenderBlush,
	"lawngreen":            ColorLawnGreen,
	"lemonchiffon":         ColorLemonChiffon,
	"lightblue":            ColorLightBlue,
	"lightcoral":           ColorLightCoral,
	"lightcyan":            ColorLightCyan,
	"lightgoldenrodyellow": ColorLightGoldenrodYellow,
	"lightgray":            ColorLightGray,
	"lightgreen":           ColorLightGreen,
	"lightpink":            ColorLightPink,
	"lightsalmon":          ColorLightSalmon,
	"lightseagreen":        ColorLightSeaGreen,
	"lightskyblue":         ColorLightSkyBlue,
	"lightslategray":       ColorLightSlateGray,
	"lightsteelblue":       ColorLightSteelBlue,
	"lightyellow":          ColorLightYellow,
	"limegreen":            ColorLimeGreen,
	"linen":                ColorLinen,
	"mediumaquamarine":     ColorMediumAquamarine,
	"mediumblue":           ColorMediumBlue,
	"mediumorchid":         ColorMediumOrchid,
	"mediumpurple":         ColorMediumPurple,
	"mediumseagreen":       ColorMediumSeaGreen,
	"mediumslateblue":      ColorMediumSlateBlue,
	"mediumspringgreen":    ColorMediumSpringGreen,
	"mediumturquoise":      ColorMediumTurquoise,
	"mediumvioletred":      ColorMediumVioletRed,
	"midnightblue":         ColorMidnightBlue,
	"mintcream":            ColorMintCream,
	"mistyrose":            ColorMistyRose,
	"moccasin":             ColorMoccasin,
	"navajowhite":          ColorNavajoWhite,
	"oldlace":              ColorOldLace,
	"olivedrab":            ColorOliveDrab,
	"orange":               ColorOrange,
	"orangered":            ColorOrangeRed,
	"orchid":               ColorOrchid,
	"palegoldenrod":        ColorPaleGoldenrod,
	"palegreen":            ColorPaleGreen,
	"paleturquoise":        ColorPaleTurquoise,
	"palevioletred":        ColorPaleVioletRed,
	"papayawhip":           ColorPapayaWhip,
	"peachpuff":            ColorPeachPuff,
	"peru":                 ColorPeru,
	"pink":                 ColorPink,
	"plum":                 ColorPlum,
	"powderblue":           ColorPowderBlue,
	"rebeccapurple":        ColorRebeccaPurple,
	"rosybrown":            ColorRosyBrown,
	"royalblue":            ColorRoyalBlue,
	"saddlebrown":          ColorSaddleBrown,
	"salmon":               ColorSalmon,
	"sandybrown":           ColorSandyBrown,
	"seagreen":             ColorSeaGreen,
	"seashell":             ColorSeashell,
	"sienna":               ColorSienna,
	"skyblue":              ColorSkyblue,
	"slateblue":            ColorSlateBlue,
	"slategray":            ColorSlateGray,
	"snow":                 ColorSnow,
	"springgreen":          ColorSpringGreen,
	"steelblue":            ColorSteelBlue,
	"tan":                  ColorTan,
	"thistle":              ColorThistle,
	"tomato":               ColorTomato,
	"turquoise":            ColorTurquoise,
	"violet":               ColorViolet,
	"wheat":                ColorWheat,
	"whitesmoke":           ColorWhiteSmoke,
	"yellowgreen":          ColorYellowGreen,
	"grey":                 ColorGray,
	"dimgrey":              ColorDimGray,
	"darkgrey":             ColorDarkGray,
	"darkslategrey":        ColorDarkSlateGray,
	"lightgrey":            ColorLightGray,
	"lightslategrey":       ColorLightSlateGray,
	"slategrey":            ColorSlateGray,
}

// ColorFromName returns the Color that corresponds to the given color name
func ColorFromName(name string) Color {
	return colorNames[name]
}
