package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/mitchellh/go-homedir"

	"github.com/manifoldco/promptui"
	"github.com/urfave/cli"
)

type Options struct {
	DryRun  bool
	Restore bool
}

func main() {
	// コマンドツールの基本情報
	app := cli.NewApp()
	app.Name = "fruorepo"
	app.Usage = "We enjoy repository."
	app.Version = "0.1.1"
	app.UsageText = "fruorepo [options]"
	app.Author = "k-masatany"
	app.Email = "sonntag902@gmail.com"
	app.Copyright = `
	MIT License

	Copyright (c) 2018 Kensuke Masatani

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE.
`
	opt := new(Options)
	app.Action = func(c *cli.Context) error {
		opt.DryRun = c.Bool("dry-run")
		opt.Restore = (c.Bool("restore") || c.Bool("r"))
		run(opt)
		return nil
	}

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "dry-run, d",
			Usage: "Run, But Not Change Repository.",
		},
		cli.BoolFlag{
			Name:  "restore, r",
			Usage: "Restore To The State Just Before The Repository.",
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(opt *Options) {
	f := new(Fruorepo)
	f.SelectRepository()
	f.PrintRepositoryOverview()

	accept := confirm("Do you want to modify this repository?")
	if accept != "y" {
		MessageAndDie("Exiting on user command")
	}

	// FetchLabels
	labels, err := f.FetchLabels()
	if err != nil {
		MessageAndDie(err.Error())
	}

	// DeleteLabels
	for _, label := range labels {
		err := f.DeleteLabel(label, opt)
		if err != nil {
			MessageAndDie(err.Error())
		}
	}

	// CreateLabels
	labelSet := []map[string]string{}
	labelSet = append(labelSet, map[string]string{"name": ":scroll:クエスト中", "color": "17139C"})                // WIP
	labelSet = append(labelSet, map[string]string{"name": ":mag:鑑定待ち", "color": "5FCC9C"})                    // レビュー待ち
	labelSet = append(labelSet, map[string]string{"name": ":moneybag:クエスト達成", "color": "FFA3AC"})             // 作業終了
	labelSet = append(labelSet, map[string]string{"name": ":speech_balloon:追加クエスト", "color": "FFCD19"})       // 修正依頼
	labelSet = append(labelSet, map[string]string{"name": ":crossed_swords:討伐中", "color": "6EC4FF"})          // WIP（バグ取り）
	labelSet = append(labelSet, map[string]string{"name": ":+1:討伐済み", "color": "6EC4FF"})                     // バグ取り完了
	labelSet = append(labelSet, map[string]string{"name": ":dragon:ドラゴン", "color": "E43A19"})                 // バグ：優先度高
	labelSet = append(labelSet, map[string]string{"name": ":sparkles:スキルアップ", "color": "ECFEFF"})             // 機能追加
	labelSet = append(labelSet, map[string]string{"name": ":face_with_head_bandage:援軍要請", "color": "B30753"}) // Help

	// Backup/Restore
	path, err := homedir.Dir()
	if err != nil {
		MessageAndDie(err.Error())
	}
	path += "/.fruorepo"
	if os.MkdirAll(path, 0755) != nil {
		MessageAndDie(err.Error())
	}
	path += "/restore"

	if opt.Restore == true {
		labelSet, err = restoreLabelset(path)
		if err != nil {
			MessageAndDie(err.Error())
		}
	}
	err = backupLabelset(path, labels)
	if err != nil {
		MessageAndDie(err.Error())
	}

	for _, label := range labelSet {
		err := f.CreateLabel(label["name"], label["color"], opt)
		if err != nil {
			MessageAndDie(err.Error())
		}
	}

	fmt.Println("Repository has been changed.")
	fmt.Println("Have a nice day!!")
}

// MessageAndDie
func MessageAndDie(s string) {
	fmt.Println(s)
	os.Exit(-1)
}

// bool prompt wrapper
func confirm(message string) string {
	prompt := promptui.Prompt{
		Label:     message,
		IsConfirm: true,
	}
	ret, err := prompt.Run()
	if err != nil {
		MessageAndDie(fmt.Sprintf("Prompt failed %v\n", err))
	}

	return ret
}

// backup labelset
func backupLabelset(path string, labels []*github.Label) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, label := range labels {
		writer.WriteString(fmt.Sprintf("%s, %s\n", *label.Name, *label.Color))
	}
	writer.Flush()

	return nil
}

// restore labelset
func restoreLabelset(path string) ([]map[string]string, error) {
	labelSet := []map[string]string{}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		params := strings.Split(scanner.Text(), ", ")
		labelSet = append(labelSet, map[string]string{"name": params[0], "color": params[1]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return labelSet, nil
}
