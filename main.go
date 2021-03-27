package main

import (
	"fmt"
	"log"
	"os"

	_ "embed"

	"github.com/gonejack/mhtml-to-epub/cmd"
	"github.com/spf13/cobra"
)

var (
	//go:embed cover.png
	defaultCover []byte

	cover   *string
	title   *string
	author  *string
	output  *string
	verbose = false

	prog = &cobra.Command{
		Use:   "mhtml-to-epub [-o output] [--title title] [--cover cover] *.mht",
		Short: "Command line tool for converting mhtml to epub.",
		Run: func(c *cobra.Command, args []string) {
			err := run(c, args)
			if err != nil {
				log.Fatal(err)
			}
		},
	}
)

func init() {
	log.SetOutput(os.Stdout)

	prog.Flags().SortFlags = false
	prog.PersistentFlags().SortFlags = false

	cover = prog.PersistentFlags().StringP(
		"cover",
		"",
		"",
		"epub cover image",
	)
	title = prog.PersistentFlags().StringP(
		"title",
		"",
		"MHTML",
		"epub title",
	)
	author = prog.PersistentFlags().StringP(
		"author",
		"",
		"MHTML to Epub",
		"epub author",
	)
	output = prog.PersistentFlags().StringP(
		"output",
		"o",
		"output.epub",
		"output filename",
	)
	prog.PersistentFlags().BoolVarP(
		&verbose,
		"verbose",
		"v",
		false,
		"verbose",
	)
}

func run(c *cobra.Command, args []string) error {
	_, err := os.Stat(*output)
	if !os.IsNotExist(err) {
		return fmt.Errorf("output file %s already exist", *output)
	}

	exec := cmd.MHTMLToEpub{
		DefaultCover: defaultCover,

		Cover:   *cover,
		Title:   *title,
		Author:  *author,
		Verbose: verbose,
	}

	return exec.Run(args, *output)
}

func main() {
	_ = prog.Execute()
}
