package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Taboo struct {
	Word        string   `json:"word"`
	BannedWords []string `json:"banned_words"`
}

func NewTaboo(word string, bannedWords []string) *Taboo {
	return &Taboo{
		Word:        word,
		BannedWords: bannedWords,
	}
}

func TaboosFromJson() []Taboo {
	file, err := os.Open("taboos.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	byteValue, _ := ioutil.ReadAll(file)

	var taboos []Taboo
	json.Unmarshal(byteValue, &taboos)
	return taboos
}
