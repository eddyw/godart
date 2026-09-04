package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
)

func main() {
	opts := &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: drop}
	h := slog.NewTextHandler(os.Stdout, opts)
	lg := slog.New(h)
	lg.Info("start", "count", 3, "name", "gart")
	lg.Debug("low", slog.Int("n", 1))
	lg.Warn("careful")
	lg.Error("bad", slog.String("why", "test"))

	group := lg.With("service", "demo").WithGroup("req")
	group.Info("inside", slog.Bool("ok", true), slog.Float64("ratio", 0.5))

	j := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	j.LogAttrs(context.Background(), slog.LevelInfo, "json", slog.Any("list", []int{1, 2}))

	fmt.Println("levels:", slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError)
	var lv slog.LevelVar
	lv.Set(slog.LevelWarn)
	fmt.Println("var:", lv.Level(), h.Enabled(context.Background(), slog.LevelDebug))

	v := slog.StringValue("text")
	fmt.Println("value:", v.Kind(), v.String(), v.Equal(slog.StringValue("text")))
	g := slog.GroupValue(slog.Int("a", 1), slog.Int("b", 2))
	fmt.Println("group:", g.Kind(), len(g.Group()), g.String())
	fmt.Println("any:", slog.AnyValue(42).Kind(), slog.AnyValue(1.5).Kind(), slog.AnyValue(nil).Kind())

	a := slog.Attr{Key: "k", Value: slog.Int64Value(7)}
	fmt.Println("attr:", a.String(), a.Value.Int64(), a.Equal(slog.Int64("k", 7)))

	// The default handler writes through the standard logger of the package
	// log. The flag set of 0 leaves the stamp of the clock out.
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
	slog.Info("through log", "n", 5)
	slog.Default().Warn("default logger")
	slog.SetLogLoggerLevel(slog.LevelError)
	slog.Info("hidden")
	slog.Error("shown")
}

// drop removes the stamp of the clock, thus the 2 runs write the same text.
func drop(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}
