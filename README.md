# CLOB Bubble Tea Component

A simple, reusable central limit order book (CLOB) component for [Bubble Tea](https://github.com/charmbracelet/bubbletea) applications.

![CLOB Component](https://raw.githubusercontent.com/charmbracelet/bubbletea/master/examples/clob/clob.gif)

## Installation

Since this is a local component, you can just copy the `clob` directory into your project.

## Usage

Here's a simple example of how to use the `clob` component in your Bubble Tea application:

```go
package main

import (
	"log"
	"os"

	"your/module/path/clob"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// mainModel represents the state of our TUI application.
type mainModel struct {
	clob   clob.Model
	width  int
	height int
}

// InitialModel creates the initial state of the application model.
func InitialModel() mainModel {
	m := mainModel{
		clob: clob.New(),
	}
	m.clob.Asks = []clob.Order{
		{Price: 100, Volume: 5},
		{Price: 101, Volume: 10},
		{Price: 102, Volume: 20},
	}
	m.clob.Bids = []clob.Order{
		{Price: 99, Volume: 1},
		{Price: 98, Volume: 20},
		{Price: 97, Volume: 40},
	}

	return m
}

// ... (rest of your Bubble Tea application)
```

## Customization

You can customize the appearance and behavior of the `clob` component by setting the fields on the `clob.Model`.

### Dimensions

You can set the width and height of the component by passing a `clob.ViewOptions` struct to the `ViewWithOptions` function.

```go
func (m mainModel) View() string {
	return m.clob.ViewWithOptions(clob.ViewOptions{Width: m.width / 2, Height: m.height / 2})
}
```

### Styling

You can override the default colors by setting the `StyleOnBid`, `StyleOnAsk`, and `StyleOffBar` fields on the `clob.Model`.

```go
func InitialModel() mainModel {
	m := mainModel{
		clob: clob.New(),
	}

	// Override default styles
	m.clob.StyleOnBid = lipgloss.NewStyle().
		Foreground(lipgloss.Color("228")).
		Background(lipgloss.Color("64"))
	m.clob.StyleOnAsk = lipgloss.NewStyle().
		Foreground(lipgloss.Color("228")).
		Background(lipgloss.Color("164"))

	// ... (rest of your model initialization)

	return m
}
```

### Spacing

You can adjust the spacing between the bid and ask columns by setting the `Spacing` field on the `clob.Model`.

```go
func InitialModel() mainModel {
	m := mainModel{
		clob: clob.New(),
	}

	m.clob.Spacing = 4

	// ... (rest of your model initialization)

	return m
}
```

## API Reference

### `clob.New()`

Creates a new `clob.Model` with default styles.

### `(m *Model) ViewWithOptions(opts ViewOptions)`

Renders the CLOB with the given options.

### `clob.Model`

*   `OrderBook`: The data for the order book.
*   `Spacing`: The space between the bid and ask columns.
*   `StyleOffBar`: The style for the "off" part of the volume bar.
*   `StyleOnBid`: The style for the bid volume bar.
*   `StyleOnAsk`: The style for the ask volume bar.
