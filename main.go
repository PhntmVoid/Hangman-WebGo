package main

import (
	"bufio"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	chosenWord     string
	guessedLetters []string
	attemptsLeft   = 10
	tmpl           *template.Template
)

// Initialisation et gestion des routes
func main() {
	var err error
	tmpl, err = template.ParseGlob("./templates/*.html")
	if err != nil {
		fmt.Printf("Erreur : %s\n", err.Error())
		return
	}

	http.HandleFunc("/accueil", accueilHandler)
	http.HandleFunc("/rules", rulesHandler)
	http.HandleFunc("/hangman", hangmanHandler)
	http.HandleFunc("/guess", guessHandler)
	http.HandleFunc("/result", resultHandler)

	rootDoc, _ := os.Getwd()
	fileserver := http.FileServer(http.Dir(rootDoc + "/assets"))
	http.Handle("/static/", http.StripPrefix("/static/", fileserver))

	fmt.Println("Serveur lancé sur : http://localhost:8080/accueil")
	http.ListenAndServe(":8080", nil)
}

func accueilHandler(w http.ResponseWriter, r *http.Request) {
	resetGame()
	tmpl.ExecuteTemplate(w, "accueil", nil)
}

func rulesHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "rules", nil)
}

// Gestion des parties
func hangmanHandler(w http.ResponseWriter, r *http.Request) {
	difficulty := r.URL.Query().Get("difficulty")

	if chosenWord == "" {
		chosenWord, _ = pickRandomWord(difficulty)
		fmt.Println("Mot choisi :", chosenWord)
		revealLetters(chosenWord, difficulty)
	}

	if isWordGuessed(chosenWord, guessedLetters) || attemptsLeft <= 0 {
		http.Redirect(w, r, "/result", http.StatusSeeOther)
		return
	}

	data := struct {
		ChosenWord     string
		GuessedLetters []string
		AttemptsLeft   int
		Difficulty     string
	}{
		ChosenWord:     maskWord(chosenWord, guessedLetters),
		GuessedLetters: guessedLetters,
		AttemptsLeft:   attemptsLeft,
		Difficulty:     difficulty,
	}

	if err := tmpl.ExecuteTemplate(w, "game", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Won        bool
		Lost       bool
		ChosenWord string
	}{
		Won:        isWordGuessed(chosenWord, guessedLetters),
		Lost:       attemptsLeft <= 0,
		ChosenWord: chosenWord,
	}

	if err := tmpl.ExecuteTemplate(w, "result", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func guessHandler(w http.ResponseWriter, r *http.Request) {
	letter := strings.ToUpper(r.FormValue("letter"))
	if letter == "" {
		http.Error(w, "Pas de lettre reçue", http.StatusBadRequest)
		return
	}

	if !isLetterAlreadyGuessed(letter) {
		guessedLetters = append(guessedLetters, letter)
		if !isLetterInWord(letter, chosenWord) {
			attemptsLeft--
		}
	}

	http.Redirect(w, r, "/hangman?difficulty="+r.URL.Query().Get("difficulty"), http.StatusSeeOther)
}

// Fonctions auxiliaires
func pickRandomWord(difficulty string) (string, error) {
	filePath := map[string]string{
		"facile":    "assets/ressources/mot.txt",
		"moyen":     "assets/ressources/mot_2.txt",
		"difficile": "assets/ressources/mot_3.txt",
	}[difficulty]

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	words := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	rand.Seed(time.Now().UnixNano())
	return words[rand.Intn(len(words))], nil
}

func revealLetters(word, difficulty string) {
	vowels := "AEIOU"
	wordLength := len(word)

	numLettersToReveal := map[string]int{"facile": 2, "moyen": 1, "difficile": 0}[difficulty]
	if wordLength <= 4 {
		for _, char := range word {
			if strings.ContainsRune(vowels, char) {
				guessedLetters = append(guessedLetters, string(char))
				return
			}
		}
	}

	revealed := map[rune]bool{}
	for i := 0; i < numLettersToReveal; i++ {
		for {
			randIndex := rand.Intn(wordLength)
			letter := rune(word[randIndex])
			if !revealed[letter] {
				revealed[letter] = true
				guessedLetters = append(guessedLetters, string(letter))
				break
			}
		}
	}
}

func maskWord(word string, guessedLetters []string) string {
	var masked strings.Builder
	for _, char := range word {
		if contains(guessedLetters, string(char)) {
			masked.WriteRune(char)
		} else {
			masked.WriteRune('_')
		}
		masked.WriteRune(' ')
	}
	return masked.String()
}

func isWordGuessed(word string, guessedLetters []string) bool {
	for _, char := range word {
		if !contains(guessedLetters, string(char)) {
			return false
		}
	}
	return true
}

func isLetterInWord(letter, word string) bool {
	return strings.Contains(word, letter)
}

func isLetterAlreadyGuessed(letter string) bool {
	return contains(guessedLetters, letter)
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func resetGame() {
	chosenWord = ""
	guessedLetters = nil
	attemptsLeft = 10
}
