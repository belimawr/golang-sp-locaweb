package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/edwardmartinsjr/golang-sp-locaweb/floresta/config"
	"github.com/edwardmartinsjr/golang-sp-locaweb/floresta/store"
	"github.com/gocarina/gocsv"
	. "github.com/jbrukh/bayesian"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", config.PostgresConnectionString())

	if err != nil {
		log.Printf("Error connecting to database: %s", err.Error())
	}

	store := store.Store{DB: db}

	for _, a := range store.ReadArvores(10) {
		fmt.Printf("%#v\n", a)
	}

	classifier()
}

var dictionary = []*Dictionary{}
var searchDictionary = map [string]*Dictionary{}

func classifier() {

	// I - Definição das classes
	const (
		Good    Class = "Good"    /* 0 */
		Neutral Class = "Neutral" /* 1 */
		Bad     Class = "Bad"     /* 2 */
	)
	classifier := NewClassifier(Good, Neutral, Bad)

	// II - Treinamento (dicionário polarizado)
	goodStuff, neutralStuff, badStuff := loadDict("./oplexicon_v3.0/lexico_v3.0.txt")
	fmt.Println("[0] Good - [1] Neutral - [2] Bad")

	// III - Aprendizado
	classifier.Learn(goodStuff, Good)
	classifier.Learn(neutralStuff, Neutral)
	classifier.Learn(badStuff, Bad)

	// IV - Coleta de Dados & V - Split dos atributos
	tweets := loadTweets("./twitter.json")

	// VI - Classificação
	for i, item := range tweets {
		scores, likely, _ := classifier.ProbScores(item.Term)
		//fmt.Println(item.Term)
		
		for _, term :=range item.Term{
			if x, ok := searchDictionary[term]; ok {
					fmt.Println(x.Attribute + " - "+ x.Type)
				}
		}

		fmt.Println(i, scores, likely)
	}


}

func loadDict(file string) ([]string, []string, []string) {
	//Realiza a leitura do arquivo CSV
	dictionaryFile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer dictionaryFile.Close()

	//	dictionary := []*Dictionary{}
	if err := gocsv.UnmarshalFile(dictionaryFile, &dictionary); err != nil { // Load dictionary from file
		panic(err)
	}

	for _, d := range dictionary{
		searchDictionary[d.Attribute] = d
	}

	goodStuff := make([]string, len(dictionary))
	neutralStuff := make([]string, len(dictionary))
	badStuff := make([]string, len(dictionary))

	var goodStuffCount int
	var neutralStuffCount int
	var badStuffCount int
	for i, item := range dictionary {
		switch item.Class {
		case "1":
			goodStuff[i] = item.Attribute
			goodStuffCount++
		case "0":
			neutralStuff[i] = item.Attribute
			neutralStuffCount++
		case "-1":
			badStuff[i] = item.Attribute
			badStuffCount++
		}
	}

	/* Remove atributos em branco do array */
	goodStuffAdjusted := make([]string, goodStuffCount)
	goodStuffCount = 0
	for _, item := range goodStuff {
		if item != "" {
			goodStuffAdjusted[goodStuffCount] = item
			goodStuffCount++
		}
	}

	neutralStuffAdjusted := make([]string, neutralStuffCount)
	neutralStuffCount = 0
	for _, item := range neutralStuff {
		if item != "" {
			neutralStuffAdjusted[neutralStuffCount] = item
			neutralStuffCount++
		}
	}

	badStuffAdjusted := make([]string, badStuffCount)
	badStuffCount = 0
	for _, item := range badStuff {
		if item != "" {
			badStuffAdjusted[badStuffCount] = item
			badStuffCount++
		}
	}

	return goodStuffAdjusted, neutralStuffAdjusted, badStuffAdjusted
}

func loadTweets(file string) []TweetListSplitted {
	//Realiza a leitura do arquivo json
	raw, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var tweetList []TweetList

	//Unmarshal do conteúdo do arquivo json para um tipo struct TweetList
	json.Unmarshal(raw, &tweetList)

	tweetListSplitted := make([]TweetListSplitted, len(tweetList))
	for i, item := range tweetList {

		//Primeiro vamos fazer através da função Fields o split da sentença por espaços
		tweet := strings.Fields(item.Tweet)
		features := make([]string, len(tweet))

		var count int
		for j, termTweet := range tweet {
			//Estamos considerando apenas palavras maiores que três caracteres para serem consideradas como atributos válidos
			//Utilizamos rune para prevenir caracteres especiais, acentos, caracteres asiaticos e também emogis
			if len([]rune(termTweet)) >= 3 {
				features[j] = strings.ToLower(termTweet)
				count++
			}
		}

		tweetListSplitted[i].Term = make([]string, count)
		count = 0
		for k, termTweetClassifier := range features {
			if features[k] != "" {
				tweetListSplitted[i].Term[count] = termTweetClassifier
				count++
			}
		}

	}

	return tweetListSplitted
}

//TweetList : Base de tweets
type TweetList struct {
	Tweet string
}

//TweetListSplitted : Lista dos atributos dos tweets a serem classificados
type TweetListSplitted struct {
	Term []string
}

//Dictionary : Léxico de sentimento para a língua portuguesa - http://ontolp.inf.pucrs.br/Recursos/downloads-OpLexicon.php
type Dictionary struct {
	Attribute          string /* ATRIBUTO */
	Type               string /* NLP */
	Class              string /* -1 - NEGATIVO; 0 - NEUTRO; 1 - POSITIVO  */
	ClassificationType string /* A - AUTOMATICA; M - MANUAL */

}


