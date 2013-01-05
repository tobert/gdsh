package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
)

func placeholderFile() string {
	return path.Join(os.Getenv("HOME"), ".gdsh", ".nssh-next.json")
}

func loadPlaceholder() (ph map[string]string) {
	jsonBytes, err := ioutil.ReadFile(placeholderFile())
	if os.IsNotExist(err) {
		// placeholder doesn't exist, return emtpy data structure
		// that will get persisted before the first item is returned
		ph = map[string]string{}
		return
	} else if err != nil {
		log.Fatal("Could not read placeholder: %s", err)
	}

	err = json.Unmarshal(jsonBytes, &ph)
	if err != nil {
		log.Fatal("Could parse placeholder: %s", err)
	}

	return
}

func savePlaceholder(ph map[string]string) {
	jsonBytes, err := json.Marshal(&ph)
	if err == nil {
		ioutil.WriteFile(placeholderFile(), jsonBytes, 0644)
	} else {
		log.Fatal("BUG: could not persist placeholder data: ", err)
	}
}

func updatePlaceholder(listName string, nodeName string) {
	ph := loadPlaceholder()
	ph[listName] = nodeName
	savePlaceholder(ph)
}

func nextNode(listName string) (node Node) {
	list := loadListByName(listName)
	ph := loadPlaceholder()

	if previous, ok := ph[listName]; ok {
		for i, node := range list {
			if i == len(list)-1 {
				log.Fatal("Already at last node in list '%s'\n", listName)
			} else if node.Address == previous {
				ph[listName] = list[i+1].Address
				savePlaceholder(ph)
				return list[i+1]
			}
		}
	}

	// no placeholder found, create one
	updatePlaceholder(listName, list[0].Address)
	return list[0]
}

func resetNode(dshListName string) {
	ph := loadPlaceholder()
	delete(ph, dshListName)
	savePlaceholder(ph)
}

// vim: ts=4 sw=4 noet tw=120 softtabstop=4
