package app

import (
	"gart.dev/ui"
	"gart.dev/ui/material"
)

// This file holds the widgets of ui/core/ui of the sample.

// iconBox gives the 40 by 40 button of the sample: a border of grey1, a corner
// of 8, and 1 icon of the color onSurface. A blur draws the content behind it.
func iconBox(icon string, blur bool, onTap func()) ui.Node {
	box := ui.DecoratedBox{
		Decoration: ui.Decoration{
			Border: &ui.BorderSide{Color: ui.P(colorGrey1)},
			Radius: ui.P(ui.AllCorners(8)),
		},
		Child: material.InkWell{
			BorderRadius: ui.P(ui.AllCorners(8)),
			OnTap:        onTap,
			Child: ui.Center{Child: ui.Icon{
				Icon:  icon,
				Size:  ui.P(24.0),
				Color: ui.P(ui.Theme().Colors.OnSurface),
			}},
		},
	}
	kids := ui.Kids(box)
	if blur {
		kids = ui.Kids(
			ui.ClipRect{Child: ui.Blur{
				SigmaX: 2,
				SigmaY: 2,
				Child:  ui.SizedBox{Height: ui.P(40.0), Width: ui.P(40.0)},
			}},
			box,
		)
	}
	return ui.SizedBox{
		Height: ui.P(40.0),
		Width:  ui.P(40.0),
		Child:  ui.Stack{Fit: ui.P("expand"), Children: kids},
	}
}

// backButton gives the arrow of the top left corner.
func backButton(blur bool, onTap func()) ui.Node {
	return iconBox("arrow_back", blur, onTap)
}

// homeButton opens the home screen.
func (a *App) homeButton(blur bool) ui.Node {
	return iconBox("home_outlined", blur, func() { a.replace(route{name: screenHome}) })
}

// logoutButton drops the token and it opens the login screen.
func (a *App) logoutButton() ui.Node {
	return iconBox("logout", false, a.logout)
}

// customCheckbox gives the round box of the activity list.
func customCheckbox(value bool, onChanged func(bool)) ui.Node {
	fill := ui.Color(0)
	if value {
		fill = ui.Theme().Colors.Primary
	}
	return material.InkResponse{
		Radius: ui.P(24.0),
		OnTap:  func() { onChanged(!value) },
		Child: ui.DecoratedBox{
			Decoration: ui.Decoration{
				Radius: ui.P(ui.AllCorners(24)),
				Border: &ui.BorderSide{Color: ui.P(colorGrey3)},
			},
			Child: material.Material{
				BorderRadius: ui.P(ui.AllCorners(24)),
				Color:        ui.P(fill),
				Child: ui.SizedBox{
					Width:  ui.P(24.0),
					Height: ui.P(24.0),
					Child: ui.Visibility{
						Visible: ui.P(value),
						Child: ui.Icon{
							Icon:  "check",
							Size:  ui.P(14.0),
							Color: ui.P(ui.Theme().Colors.OnPrimary),
						},
					},
				},
			},
		},
	}
}

// errorIndicator gives the message of a read that failed, and a button that
// runs the read again.
func errorIndicator(title, label string, onPressed func()) ui.Node {
	onError := ui.Theme().Colors.OnError
	return ui.Column{
		MainAxisAlignment:  ui.P("center"),
		CrossAxisAlignment: ui.P("center"),
		Children: ui.Kids(
			ui.IntrinsicWidth{Child: ui.Padding{
				Padding: ui.AllSides(8),
				Child: ui.Center{Child: ui.Row{Children: ui.Kids(
					ui.Icon{Icon: "error_outline", Color: ui.P(onError)},
					ui.SizedBox{Width: ui.P(10.0)},
					ui.Text{Data: title, Style: &ui.TextStyle{Color: ui.P(onError)}},
				)}},
			}},
			ui.SizedBox{Height: ui.P(10.0)},
			material.FilledButton{
				OnPressed: onPressed,
				Style: &ui.ButtonStyle{
					BackgroundColor: &ui.StateColor{Default: ui.P(colorRed1)},
					ForegroundColor: &ui.StateColor{Default: ui.P(colorWhite)},
				},
				Child: ui.Text{Data: label},
			},
		),
	}
}

// searchBar gives the row of the search of the results and of the form.
//
// An empty config gives the line "Search destination". The config that the
// user filled gives the continent, the 2 dates, and the count of the guests.
func (a *App) searchBar(query string, onTap func()) ui.Node {
	var inside ui.Node
	if query == "" {
		inside = ui.Row{
			CrossAxisAlignment: ui.P("center"),
			MainAxisAlignment:  ui.P("start"),
			Children: ui.Kids(
				ui.Icon{Icon: "search"},
				ui.SizedBox{Width: ui.P(12.0)},
				ui.Expanded{Child: ui.Text{
					Data:      textSearchDestination,
					TextAlign: ui.P("start"),
					Style:     hintStyle(),
				}},
			),
		}
	} else {
		inside = ui.Text{
			Data:      query,
			TextAlign: ui.P("center"),
			Style:     ui.Theme().Text.BodyLarge,
		}
	}
	return ui.Row{Children: ui.Kids(
		ui.Expanded{Child: material.InkWell{
			BorderRadius: ui.P(ui.AllCorners(16)),
			OnTap:        onTap,
			Child: ui.Container{
				Height: ui.P(64.0),
				Decoration: &ui.Decoration{
					Border: &ui.BorderSide{Color: ui.P(colorGrey1)},
					Radius: ui.P(ui.AllCorners(16)),
				},
				Child: ui.Padding{
					Padding: ui.Sides(paddingHorizontal, 0),
					Child:   ui.Align{Alignment: ui.P("centerLeft"), Child: inside},
				},
			},
		}},
		ui.SizedBox{Width: ui.P(10.0)},
		a.homeButton(false),
	)}
}

// tagChipSmall gives the chip of the results grid. The 2 colors come from the
// theme of the application.
func tagChipSmall(name string) ui.Node {
	chip, onChip := chipColors()
	return tagChip(name, 10, 20, chip, onChip)
}

// tagChip gives 1 tag of a destination: an icon, the name, and a blur behind
// the 2.
func tagChip(name string, fontSize, height float64, chip, onChip ui.Color) ui.Node {
	return ui.ClipRRect{
		BorderRadius: ui.P(ui.AllCorners(height / 2)),
		Child: ui.Blur{
			SigmaX: 3,
			SigmaY: 3,
			Child: ui.DecoratedBox{
				Decoration: ui.Decoration{Color: ui.P(chip)},
				Child: ui.SizedBox{
					Height: ui.P(height),
					Child: ui.Padding{
						Padding: ui.Sides(6, 0),
						Child: ui.Row{
							CrossAxisAlignment: ui.P("center"),
							MainAxisSize:       ui.P("min"),
							Children: ui.Kids(
								ui.Icon{
									Icon:  tagIcon(name),
									Color: ui.P(onChip),
									Size:  ui.P(fontSize),
								},
								ui.SizedBox{Width: ui.P(4.0)},
								ui.Text{
									Data:      name,
									TextAlign: ui.P("center"),
									Style: &ui.TextStyle{
										Family:     fontOpenSans,
										Weight:     "w500",
										Variations: weight(500),
										Size:       ui.P(fontSize),
										Color:      ui.P(onChip),
									},
								},
							),
						},
					},
				},
			},
		},
	}
}

// tagIcon names the icon of 1 tag. The sample holds the same table in
// ui/core/ui/tag_chip.dart.
func tagIcon(name string) string {
	switch name {
	case "Adventure sports":
		return "kayaking_outlined"
	case "Beach":
		return "beach_access_outlined"
	case "City":
		return "location_city_outlined"
	case "Cultural experiences":
		return "museum_outlined"
	case "Foodie", "Food tours":
		return "restaurant"
	case "Hiking":
		return "hiking"
	case "Historic":
		return "menu_book_outlined"
	case "Island", "Coastal", "Lake", "River":
		return "water"
	case "Luxury":
		return "attach_money_outlined"
	case "Mountain", "Wildlife watching":
		return "landscape_outlined"
	case "Nightlife":
		return "local_bar_outlined"
	case "Off-the-beaten-path":
		return "do_not_step_outlined"
	case "Romantic":
		return "favorite_border_outlined"
	case "Rural":
		return "agriculture_outlined"
	case "Secluded":
		return "church_outlined"
	case "Sightseeing":
		return "attractions_outlined"
	case "Skiing":
		return "downhill_skiing_outlined"
	case "Wine tasting":
		return "wine_bar_outlined"
	case "Winter destination":
		return "ac_unit"
	}
	return "label_outlined"
}

// toast shows 1 message over the application, exactly as the ScaffoldMessenger
// of the sample does.
//
// It runs in a goroutine of its own, because the call parks until the user or
// the timer closes the bar. A press of the button runs retry.
func toast(message, action string, retry func()) {
	go func() {
		out, err := ui.SnackBar(message, action)
		if err != nil || out != "action" || retry == nil {
			return
		}
		ui.Post(retry)
	}()
}

// busy gives the indicator of a read that runs.
func busy() ui.Node { return ui.Center{Child: material.CircularProgressIndicator{}} }

// picture gives the image of a URL, and a grey box after a failure.
func picture(url string, width, height float64, fit string) ui.Node {
	p := ui.Picture{
		Image: ui.NetworkImage(url),
		Fit:   fit,
		Error: ui.DecoratedBox{
			Decoration: ui.Decoration{Color: ui.P(colorGrey3)},
			Child:      ui.SizedBox{Width: ui.P(width), Height: ui.P(height)},
		},
	}
	if width > 0 {
		p.Width = ui.P(width)
	}
	if height > 0 {
		p.Height = ui.P(height)
	}
	return p
}
