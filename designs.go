package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/mikebell-org/go-couchdb"
)

type viewData struct {
	MapFunc    string `json:"map,omitempty"`
	ReduceFunc string `json:"reduce,omitempty"`
}

type designDocData struct {
	TypedDocumentWithMtime
	Language string               `json:"language,omitempty"`
	Views    map[string]*viewData `json:"views,omitempty"`
	Updates  map[string]string    `json:"updates,omitempty"`
}

const (
	designs_root = "designs"
)

func readFileContents(path string, contents *string) error {

	if file, err := os.Open("./" + path); err != nil {

		return err
	} else {
		file_size, err := file.Seek(0, os.SEEK_END)
		if err != nil {

			return err
		}

		buf := make([]byte, file_size)
		if _, err := file.ReadAt(buf, 0); err != nil {

			return err
		}

		*contents = string(buf)
	}

	return nil
}

func newDesignDoc() *designDocData {

	return &designDocData{Language: "javascript", Views: map[string]*viewData{}, Updates: map[string]string{}}
}

func readDesignDocsFromDisk(path string, data map[string]*designDocData) error {

	// TODO: make the update function handler more generic so it also handles
	//       list, filter, show functions, etc.

	fmt.Printf("Scanning design docs at %v\n", path)

	// read all design docs and make sure the DB is updated with the latest
	// versions of the code
	err := filepath.Walk(designs_root, func(path string, f os.FileInfo, err error) error {

		if match, _ := filepath.Match(designs_root+"/*/views/*/*.js", path); match {

			// got ourselves a view function

			parts := strings.Split(path, string(filepath.Separator))
			num_parts := len(parts)
			view_name := parts[num_parts-2]
			doc_name := fmt.Sprintf("_design/%v", parts[num_parts-4])

			if _, ok := data[doc_name]; !ok {

				fmt.Printf("Creating new design document %v\n", doc_name)
				data[doc_name] = newDesignDoc()
			}

			if _, ok := data[doc_name].Views[view_name]; !ok {

				data[doc_name].Views[view_name] = &viewData{}
			}
			view := data[doc_name].Views[view_name]

			fmt.Printf("%v/_view/%v", doc_name, view_name)
			if parts[num_parts-1] == "map.js" {

				fmt.Println(" map function")
				if err := readFileContents(path, &view.MapFunc); err != nil {

					return err
				}
			} else if parts[num_parts-1] == "reduce.js" {

				fmt.Println(" reduce function")
			}
		} else if match, err := filepath.Match(designs_root+"/*/updates/*.js", path); match {

			// found an update function

			parts := strings.Split(path, string(filepath.Separator))
			num_parts := len(parts)
			doc_name := fmt.Sprintf("_design/%v", parts[num_parts-3])
			func_name := strings.TrimSuffix(parts[num_parts-1], ".js")

			if _, ok := data[doc_name]; !ok {

				fmt.Printf("Creating new design document %v\n", doc_name)
				data[doc_name] = newDesignDoc()
			}
			fmt.Printf("%v/_view/%v update function\n", doc_name, func_name)

			var update_func string

			if err := readFileContents(path, &update_func); err != nil {

				return err
			}

			data[doc_name].Updates[func_name] = update_func
		} else if err != nil {

			return err
		}

		return nil
	})

	return err
}

// http://play.golang.org/p/0lb3Hg8nT1
func keys(m interface{}) interface{} {

	mval := reflect.ValueOf(m)
	key_type := reflect.TypeOf(m).Key()

	keys := reflect.MakeSlice(reflect.SliceOf(key_type), 0, mval.Len())

	for _, key := range mval.MapKeys() {

		keys = reflect.Append(keys, key)
	}

	return keys.Interface()
}

func (this *designDocData) update(other *designDocData) (updated bool) {

	// TODO: handle designDocData.Language
	// TODO: make the update function handling code more generic so it also
	//       handles other type of functions (list, show, filter, etc.)

	updated = false

	// compare views
	this_views := sort.StringSlice(keys(this.Views).([]string))
	this_views.Sort()

	other_views := sort.StringSlice(keys(other.Views).([]string))
	other_views.Sort()

	for _, name := range this_views {

		if _, ok := other.Views[name]; !ok {

			fmt.Printf("View %v needs to be deleted\n", name)
			delete(this.Views, name)
			updated = true

			continue
		}

		if this.Views[name].MapFunc != other.Views[name].MapFunc {

			fmt.Printf("Map funcion for view %v is out of date\n", name)
			this.Views[name].MapFunc = other.Views[name].MapFunc
			updated = true
		}

		if this.Views[name].ReduceFunc != other.Views[name].ReduceFunc {

			fmt.Printf("Reduce funcion for view %v is out of date\n", name)
			this.Views[name].ReduceFunc = other.Views[name].ReduceFunc
			updated = true
		}
	}

	for _, name := range other_views {

		if _, ok := this.Views[name]; !ok {

			fmt.Printf("View %v is missing\n", name)
			this.Views[name] = other.Views[name]
			updated = true
		}
	}

	// compare update functions
	this_updates := sort.StringSlice(keys(this.Updates).([]string))
	this_updates.Sort()

	other_updates := sort.StringSlice(keys(other.Updates).([]string))
	other_updates.Sort()

	for _, name := range this_updates {

		if _, ok := other.Updates[name]; !ok {

			fmt.Printf("Update function %v needs to be deleted\n", name)
			delete(this.Updates, name)
			updated = true

			break
		}

		if this.Updates[name] != other.Updates[name] {

			fmt.Printf("Update funcion %v is out of date\n", name)
			this.Updates[name] = other.Updates[name]
			updated = true
		}
	}

	for _, name := range other_updates {

		if _, ok := this.Updates[name]; !ok {

			fmt.Printf("Updated function %v is missing\n", name)
			this.Updates[name] = other.Updates[name]
			updated = true
		}
	}

	return
}

func syncDesignDocs(db *couchdb.CouchDB) error {

	// TODO: implement a document freezing option

	disk_data := map[string]*designDocData{}
	if err := readDesignDocsFromDisk(designs_root, disk_data); err == nil {

		db_data := newDesignDoc()

		for doc_name, document := range disk_data {

			if err := db.GetDocument(&db_data, fmt.Sprintf("%v", doc_name)); err != nil || db_data.update(document) {

				fmt.Printf("DB code of %v needs to be updated\n", doc_name)
				if success, err := db.PutDocument(db_data, doc_name); err != nil || !success.OK {

					return err
				}
			}
		}
	} else {

		return err
	}

	return nil
}
