// Box finds an object in an image with a decision model.
//
// It sends a PNG or JPEG together with two questions: a Noul asking whether
// the object is present, and a Box asking where it is. Box always returns
// coordinates, even when the object is absent, so the Noul is what says
// whether to trust them. Box needs Jev 1.14, which is what jev-latest resolves
// to.
//
// Usage:
//
//	go run . photo.jpg "the red rectangle"
//
// Set AI_PROVIDER to typesafe (default) or openrouter, with
// TYPESAFE_API_KEY or OPENROUTER_API_KEY.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joakimcarlsson/ai/decision"
	"github.com/joakimcarlsson/ai/decision/jev"
	"github.com/joakimcarlsson/ai/decision/openrouter"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %s <image.png|image.jpg> <what to find>", os.Args[0])
	}
	path, target := os.Args[1], os.Args[2]

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("cannot read %s as an image: %v", path, err)
	}
	if format != "png" && format != "jpeg" {
		log.Fatalf("unsupported image format %q (use png or jpeg)", format)
	}
	fmt.Printf("%s: %s, %dx%d\n", filepath.Base(path), format, cfg.Width, cfg.Height)

	img := decision.NewImage(format, data)
	decider := newDecider()

	resp, err := decider.Decide(
		context.Background(),
		"photo: "+img.Ref,
		map[string]decision.Question{
			"present": decision.Noul("Is " + target + " visible in the photo?"),
			"where":   decision.Box("Locate " + target + " in the photo."),
		},
		decision.WithImages(img),
	)
	if err != nil {
		log.Fatal(err)
	}

	present, _ := resp.Noul("present")
	where, _ := resp.Box("where")
	b := where.Box

	fmt.Printf("present: %.0f%%\n", present.Probability*100)
	fmt.Printf(
		"box (fractions): top=%.2f bottom=%.2f left=%.2f right=%.2f\n",
		b.Top, b.Bottom, b.Left, b.Right,
	)
	fmt.Printf(
		"box (pixels):    x=%d..%d y=%d..%d\n",
		int(b.Left*float64(cfg.Width)), int(b.Right*float64(cfg.Width)),
		int(b.Top*float64(cfg.Height)), int(b.Bottom*float64(cfg.Height)),
	)
	if where.Confidence != nil {
		fmt.Printf("confidence: %.2f\n", *where.Confidence)
	}
	if present.Probability < 0.5 {
		fmt.Println("the object is probably not in the image; ignore the box")
	}
	fmt.Printf(
		"usage: %d in / %d out tokens (model %s)\n",
		resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.Model,
	)
}

func newDecider() decision.Decider {
	switch provider := strings.ToLower(os.Getenv("AI_PROVIDER")); provider {
	case "", "typesafe":
		return jev.NewDecider(
			jev.WithAPIKey(requiredEnv("TYPESAFE_API_KEY")),
		)
	case "openrouter":
		return openrouter.NewDecider(
			jev.WithAPIKey(requiredEnv("OPENROUTER_API_KEY")),
		)
	default:
		log.Fatalf(
			"unsupported AI_PROVIDER %q (use typesafe or openrouter)",
			provider,
		)
		return nil
	}
}

func requiredEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s is required", name)
	}
	return v
}
