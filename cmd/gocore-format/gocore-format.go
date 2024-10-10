package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"
)

type Setting struct {
	Key      string
	Comments string
	Variants []Variant
}

type Variant struct {
	Commented bool
	Key       string
	Value     string
	Comment   string // The comment after the key=value pair
}

func main() {
	var (
		write    bool
		help     bool
		filename string
		in       = os.Stdin
		err      error
	)

	flag.BoolVar(&write, "w", false, "Write to file")
	flag.BoolVar(&help, "h", false, "Help")
	flag.Parse()

	if help {
		flag.PrintDefaults()
		return
	}

	args := flag.Args()

	if len(args) > 0 {
		filename = args[0]

		in, err = os.Open(filename)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer in.Close()
	}

	settings, err := readSettings(in)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	sortSettings(settings)

	if filename != "" && write {
		in.Close()

		out, err := os.Create(filename + ".tmp")
		if err != nil {
			fmt.Println("Error creating output file:", err)
			return
		}
		defer out.Close()

		if err := writeSettings(out, settings); err != nil {
			fmt.Println("Error writing file:", err)
			return
		}

		if err := os.Rename(filename+".tmp", filename); err != nil {
			fmt.Println("Error renaming file:", err)
			return
		}
	} else {
		if err := writeSettings(os.Stdout, settings); err != nil {
			fmt.Println("Error writing file:", err)
			return
		}
	}
}

func readSettings(r io.Reader) ([]*Setting, error) {
	var pendingSectionComment string

	settings := make(map[string]*Setting)

	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		item := processLine(line)

		if item == nil {
			// This is an arbitrary comment line
			line = strings.TrimSpace(line[1:])

			if pendingSectionComment == "" {
				pendingSectionComment = line
			} else {
				pendingSectionComment += "\n" + line
			}
		} else {
			rootKey := strings.Split(item.Key, ".")[0]

			setting, found := settings[rootKey]
			if !found {
				setting = &Setting{
					Key:      rootKey,
					Comments: pendingSectionComment,
				}

				pendingSectionComment = ""
			}

			setting.Variants = append(setting.Variants, *item)

			settings[rootKey] = setting
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	settingsSlice := make([]*Setting, 0, len(settings))
	for _, setting := range settings {
		settingsSlice = append(settingsSlice, setting)
	}

	return settingsSlice, nil
}

func writeSettings(w io.Writer, settings []*Setting) error {
	writer := bufio.NewWriter(w)
	defer writer.Flush()

	for _, setting := range settings {
		if setting.Comments != "" {
			_, err := writer.WriteString("# " + setting.Comments + "\n")
			if err != nil {
				return err
			}
		}

		maxKeyLength := 0

		for _, variant := range setting.Variants {

			l := len(variant.Key)
			if variant.Commented {
				l += 2
			}

			if l > maxKeyLength {
				maxKeyLength = l
			}
		}

		for _, variant := range setting.Variants {
			prefix := ""

			length := maxKeyLength

			if variant.Commented {
				prefix = "# "
				length -= 2
			}

			value := cleanMultiValues(variant.Value)

			line := fmt.Sprintf("%s%-*s = %s", prefix, length, variant.Key, value)

			if variant.Comment != "" {
				line += " # " + variant.Comment
			}

			_, err := writer.WriteString(line + "\n")
			if err != nil {
				return err
			}
		}

		_, err := writer.WriteString("\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func processLine(line string) *Variant {

	setting := &Variant{}

	if strings.HasPrefix(line, "#") {
		setting.Commented = true
		line = line[1:]
	}

	parts := strings.SplitN(line, "=", 2)

	if len(parts) == 1 {
		return nil
	}

	setting.Key = cleanKey(parts[0])

	line = strings.TrimSpace(parts[1])

	valueParts := strings.SplitN(line, "#", 2)
	setting.Value = strings.TrimSpace(valueParts[0])

	if len(valueParts) > 1 {
		setting.Comment = strings.TrimSpace(valueParts[1])
	}

	return setting
}

func cleanKey(key string) string {
	parts := strings.Split(strings.TrimSpace(key), ".")

	for i := 0; i < len(parts); i++ {
		parts[i] = strings.TrimSpace(parts[i])
	}

	return strings.Join(parts, ".")
}

func cleanMultiValues(value string) string {
	parts := strings.Split(value, "|")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	return strings.Join(parts, " | ")
}

func sortSettings(settings []*Setting) {
	sort.Slice(settings, func(i, j int) bool {
		r1, r2 := rune(settings[i].Key[0]), rune(settings[j].Key[0])
		if unicode.IsUpper(r1) != unicode.IsUpper(r2) {
			return unicode.IsUpper(r1)
		}

		return settings[i].Key < settings[j].Key
	})
}
