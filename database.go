package main

import (
	"math"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type stat struct {
	_id    bson.ObjectId
	ID     string
	Game   string
	Hours  float64
	Ignore bool
}

type icon struct {
	_id      bson.ObjectId
	Game     string
	Location string
	Color    string
}

type blacklist struct {
	_id  bson.ObjectId
	Game string
}

type setting struct {
	_id             bson.ObjectId
	ID              string
	GraphType       string
	MentionForStats bool
	IsGuild         bool
	Role            string
}

type datastore struct {
	session *mgo.Session
}

func (datastore *datastore) findOne(collectionName string, query bson.M, result interface{}) {
	data := datastore.session.Copy()
	defer data.Close()

	data.DB("").C(collectionName).Find(query).One(result)
}

func (datastore *datastore) findAll(collectionName string, query bson.M, results interface{}) {
	data := datastore.session.Copy()
	defer data.Close()

	data.DB("").C(collectionName).Find(query).All(results)
}

func (datastore *datastore) findAllSort(collectionName, sort string, query bson.M, results interface{}) {
	data := datastore.session.Copy()
	defer data.Close()

	data.DB("").C(collectionName).Find(query).Sort(sort).All(results)
}

func (datastore *datastore) insert(collectionName string, itemToIntert interface{}) {
	data := datastore.session.Copy()
	defer data.Close()

	data.DB("").C(collectionName).Insert(itemToIntert)
}

func (datastore *datastore) update(collectionName string, query, itemToUpdate bson.M) {
	data := datastore.session.Copy()
	defer data.Close()

	data.DB("").C(collectionName).Update(query, itemToUpdate)
}

func (datastore *datastore) removeAll(collectionName string, query bson.M) {
	data := datastore.session.Copy()
	defer data.Close()

	data.DB("").C(collectionName).RemoveAll(query)
}

func (datastore *datastore) itemExists(collectionName string, query bson.M) bool {
	data := datastore.session.Copy()
	defer data.Close()
	var result []interface{}
	data.DB("").C(collectionName).Find(query).All(&result)
	if len(result) == 0 {
		return false
	}
	return true
}

func (datastore *datastore) countHours(query bson.M) float64 {
	data := datastore.session.Copy()
	defer data.Close()
	var result []stat
	var totalHours float64

	data.DB("").C("gamestats").Find(query).All(&result)
	for _, item := range result {
		totalHours += item.Hours
	}
	return math.Round(totalHours)
}

func (datastore *datastore) countGames(query bson.M) int {
	data := datastore.session.Copy()
	defer data.Close()
	var result []string

	data.DB("").C("gamestats").Find(query).Distinct("game", &result)
	return len(result)
}

func setUpDB(dbName string) (*mgo.Session, *datastore) {
	session, err := mgo.Dial(dbName)
	if err != nil {
		panic(err)
	}

	genIndex := func(keys []string) mgo.Index {
		return mgo.Index{
			Key:        keys,
			Unique:     true,
			Background: false,
			Sparse:     true,
		}
	}

	statbotSession := session.Copy()
	defer statbotSession.Close()

	if err = statbotSession.DB("").C("gamestats").EnsureIndex(genIndex([]string{"name", "game", "id"})); err != nil {
		panic(err)
	}
	if err = statbotSession.DB("").C("gameicons").EnsureIndex(genIndex([]string{"location", "game", "color"})); err != nil {
		panic(err)
	}
	if err = statbotSession.DB("").C("iconblacklists").EnsureIndex(genIndex([]string{"game"})); err != nil {
		panic(err)
	}
	if err = statbotSession.DB("").C("settings").EnsureIndex(genIndex([]string{"id"})); err != nil {
		panic(err)
	}
	return session, &datastore{session: session}
}
