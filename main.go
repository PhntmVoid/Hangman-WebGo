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

type GameState struct {
	ChosenWord     string
	GuessedLetters []string
	AttemptsLeft   int
}

var (
	userGames = make(map[string]*GameState) // Map pour stocker les parties des utilisateurs
	tmpl      *template.Template
)

// Initialisation et gestion des routes
func main() {
	var err error
	tmpl, err = template.ParseGlob("./templates/*.html")
	if err != nil {
		fmt.Printf("Erreur : %s\n", err.Error())
		return
	}

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/accueil", accueilHandler)
	http.HandleFunc("/rules", rulesHandler)
	http.HandleFunc("/hangman", hangmanHandler)
	http.HandleFunc("/guess", guessHandler)
	http.HandleFunc("/result", resultHandler)

	rootDoc, _ := os.Getwd()
	fileserver := http.FileServer(http.Dir(rootDoc + "/assets"))
	http.Handle("/static/", http.StripPrefix("/static/", fileserver))

	fmt.Println("Serveur lancé sur : http://localhost:8080/login")
	http.ListenAndServe(":8080", nil)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		pseudo := strings.TrimSpace(r.FormValue("pseudo"))
		if pseudo == "" {
			tmpl.ExecuteTemplate(w, "login", struct {
				Error string
			}{"Le pseudo ne peut pas être vide."})
			return
		}

		if _, exists := userGames[pseudo]; !exists {
			userGames[pseudo] = &GameState{
				AttemptsLeft:   10,
				GuessedLetters: []string{},
			}
		}

		http.SetCookie(w, &http.Cookie{
			Name:  "pseudo",
			Value: pseudo,
			Path:  "/",
		})
		http.Redirect(w, r, "/accueil", http.StatusSeeOther)
		return
	}

	tmpl.ExecuteTemplate(w, "login", nil)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "pseudo",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func accueilHandler(w http.ResponseWriter, r *http.Request) {
	pseudo, err := getPseudoFromCookie(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	resetGame(pseudo)
	tmpl.ExecuteTemplate(w, "accueil", nil)
}

func rulesHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "rules", nil)
}

func hangmanHandler(w http.ResponseWriter, r *http.Request) {
	pseudo, err := getPseudoFromCookie(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	game := userGames[pseudo]
	if game.ChosenWord == "" {
		difficulty := r.URL.Query().Get("difficulty")
		game.ChosenWord, _ = pickRandomWord(difficulty)
		revealLetters(game.ChosenWord, difficulty, game)
	}

	if isWordGuessed(game.ChosenWord, game.GuessedLetters) || game.AttemptsLeft <= 0 {
		http.Redirect(w, r, "/result", http.StatusSeeOther)
		return
	}

	data := struct {
		ChosenWord     string
		GuessedLetters []string
		AttemptsLeft   int
		Pseudo         string
	}{
		ChosenWord:     maskWord(game.ChosenWord, game.GuessedLetters),
		GuessedLetters: game.GuessedLetters,
		AttemptsLeft:   game.AttemptsLeft,
		Pseudo:         pseudo,
	}

	if err := tmpl.ExecuteTemplate(w, "game", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	pseudo, err := getPseudoFromCookie(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	game := userGames[pseudo]
	data := struct {
		Won        bool
		Lost       bool
		ChosenWord string
		Pseudo     string
	}{
		Won:        isWordGuessed(game.ChosenWord, game.GuessedLetters),
		Lost:       game.AttemptsLeft <= 0,
		ChosenWord: game.ChosenWord,
		Pseudo:     pseudo,
	}

	if err := tmpl.ExecuteTemplate(w, "result", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func guessHandler(w http.ResponseWriter, r *http.Request) {
	pseudo, err := getPseudoFromCookie(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	game := userGames[pseudo]
	letter := strings.ToUpper(r.FormValue("letter"))
	if letter == "" {
		http.Error(w, "Pas de lettre reçue", http.StatusBadRequest)
		return
	}

	if !isLetterAlreadyGuessed(letter, game.GuessedLetters) {
		game.GuessedLetters = append(game.GuessedLetters, letter)
		if !isLetterInWord(letter, game.ChosenWord) {
			game.AttemptsLeft--
		}
	}

	http.Redirect(w, r, "/hangman", http.StatusSeeOther)
}

// Auxiliaires
func getPseudoFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("pseudo")
	if err != nil || cookie.Value == "" {
		return "", fmt.Errorf("Pas de pseudo")
	}
	return cookie.Value, nil
}

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

func resetGame(pseudo string) {
	game := userGames[pseudo]
	game.ChosenWord = ""
	game.GuessedLetters = []string{}
	game.AttemptsLeft = 10
}

func revealLetters(word, difficulty string, game *GameState) {
	vowels := "AEIOU"
	numLettersToReveal := map[string]int{"facile": 2, "moyen": 1, "difficile": 0}[difficulty]

	for i := 0; i < numLettersToReveal; i++ {
		for {
			randIndex := rand.Intn(len(word))
			letter := string(word[randIndex])
			if !contains(game.GuessedLetters, letter) && strings.Contains(vowels, letter) {
				game.GuessedLetters = append(game.GuessedLetters, letter)
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

func isLetterAlreadyGuessed(letter string, guessedLetters []string) bool {
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
