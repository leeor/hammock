package hammock

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/mikebell-org/go-couchdb"
)

type viewMapReduceFunctions struct {
	MapFunc    string `json:"map,omitempty"`
	ReduceFunc string `json:"reduce,omitempty"`
}

type designDocViews map[string]*viewMapReduceFunctions
type designDocFunctions map[string]string

type designDocument struct {
	couchdb.BasicDocumentWithMtime
	Name     string             `json:"-"`
	Language string             `json:"language,omitempty"`
	Views    designDocViews     `json:"views,omitempty"`
	Shows    designDocFunctions `json:"shows,omitempty"`
	Lists    designDocFunctions `json:"lists,omitempty"`
	Updates  designDocFunctions `json:"updates,omitempty"`
	Filters  designDocFunctions `json:"filters,omitempty"`
	Validate designDocFunctions `json:"validate_doc_update,omitempty"`
}

func readFileContents(path string, contents *string) error {

	if file, err := os.Open(path); err != nil {

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

func newDesignDocument(name string) *designDocument {

	return &designDocument{Name: name, Language: "javascript", Views: designDocViews{}, Updates: map[string]string{}}
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

func (doc *designDocument) readViews(views_root string) error {

	err := filepath.Walk(views_root, func(path string, f os.FileInfo, err error) error {

		if match, _ := filepath.Match(views_root+"/*/*.js", path); match {

			// got ourselves a view function

			parts := strings.Split(path, string(filepath.Separator))
			num_parts := len(parts)
			view_name := parts[num_parts-2]

			if _, ok := doc.Views[view_name]; !ok {

				doc.Views[view_name] = &viewMapReduceFunctions{}
			}
			view := doc.Views[view_name]

			if parts[num_parts-1] == "map.js" {

				if err := readFileContents(path, &view.MapFunc); err != nil {

					return err
				}
			} else if parts[num_parts-1] == "reduce.js" {

				if err := readFileContents(path, &view.ReduceFunc); err != nil {

					return err
				}
			}
		}

		return nil
	})

	return err
}

func (ddf designDocFunctions) readFunctions(root string) error {

	err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {

		if match, _ := filepath.Match(root+"/*.js", path); match {

			parts := strings.Split(path, string(filepath.Separator))
			num_parts := len(parts)
			func_name := strings.TrimSuffix(parts[num_parts-1], ".js")

			var f string

			if err := readFileContents(path, &f); err != nil {

				return err
			}

			ddf[func_name] = f
		}

		return nil
	})

	return err
}

func (doc *designDocument) loadFromDisk(document_root string) error {

	err := filepath.Walk(document_root, func(path string, f os.FileInfo, err error) error {

		if match, _ := filepath.Match(document_root+"/views", path); match && f.IsDir() {

			if err := doc.readViews(path); err != nil {

				return err
			}

			return filepath.SkipDir
		} else if match, _ := filepath.Match(document_root+"/shows", path); match && f.IsDir() {

			if err := doc.Shows.readFunctions(path); err != nil {

				return err
			}

			return filepath.SkipDir
		} else if match, _ := filepath.Match(document_root+"/lists", path); match && f.IsDir() {

			if err := doc.Lists.readFunctions(path); err != nil {

				return err
			}

			return filepath.SkipDir
		} else if match, _ := filepath.Match(document_root+"/updates", path); match && f.IsDir() {

			if err := doc.Updates.readFunctions(path); err != nil {

				return err
			}

			return filepath.SkipDir
		} else if match, _ := filepath.Match(document_root+"/filters", path); match && f.IsDir() {

			if err := doc.Filters.readFunctions(path); err != nil {

				return err
			}

			return filepath.SkipDir
		} else if match, _ := filepath.Match(document_root+"/validate", path); match && f.IsDir() {

			if err := doc.Validate.readFunctions(path); err != nil {

				return err
			}

			return filepath.SkipDir
		}

		return nil
	})

	return err
}

func updateDesignDocFunctions(this designDocFunctions, other designDocFunctions, func_type string) (updated bool, changes []string) {

	// compare update functions
	this_updates := sort.StringSlice(keys(this).([]string))
	this_updates.Sort()

	other_updates := sort.StringSlice(keys(other).([]string))
	other_updates.Sort()

	for _, name := range this_updates {

		if _, ok := other[name]; !ok {

			changes = append(changes, fmt.Sprintf("Function %v/%v needs to be deleted", func_type, name))
			delete(this, name)
			updated = true

			break
		}

		if this[name] != other[name] {

			changes = append(changes, fmt.Sprintf("Function %v/%v is out of date", func_type, name))
			this[name] = other[name]
			updated = true
		}
	}

	for _, name := range other_updates {

		if _, ok := this[name]; !ok {

			changes = append(changes, fmt.Sprintf("Function %v/%v is missing", func_type, name))
			this[name] = other[name]
			updated = true
		}
	}

	return
}

func (this *designDocument) update(other *designDocument) (updated bool, changes []string) {

	// TODO: handle designDocument.Language

	updated = false

	// compare views
	this_views := sort.StringSlice(keys(this.Views).([]string))
	this_views.Sort()

	other_views := sort.StringSlice(keys(other.Views).([]string))
	other_views.Sort()

	for _, name := range this_views {

		if _, ok := other.Views[name]; !ok {

			changes = append(changes, fmt.Sprintf("View %v/_view/%v needs to be deleted", this.Name, name))
			delete(this.Views, name)
			updated = true

			continue
		}

		if this.Views[name].MapFunc != other.Views[name].MapFunc {

			changes = append(changes, fmt.Sprintf("Map function for view %v/_view/%v is out of date", this.Name, name))
			this.Views[name].MapFunc = other.Views[name].MapFunc
			updated = true
		}

		if this.Views[name].ReduceFunc != other.Views[name].ReduceFunc {

			changes = append(changes, fmt.Sprintf("Reduce function for view %v/_view/%v is out of date", this.Name, name))
			this.Views[name].ReduceFunc = other.Views[name].ReduceFunc
			updated = true
		}
	}

	for _, name := range other_views {

		if _, ok := this.Views[name]; !ok {

			changes = append(changes, fmt.Sprintf("View %v/_view/%v is missing", this.Name, name))
			this.Views[name] = other.Views[name]
			updated = true
		}
	}

	if u, c := updateDesignDocFunctions(this.Shows, other.Shows, fmt.Sprintf("%v/_shows", this.Name)); u {

		updated = true
		changes = append(changes, c...)
	}

	if u, c := updateDesignDocFunctions(this.Lists, other.Lists, fmt.Sprintf("%v/_lists", this.Name)); u {

		updated = true
		changes = append(changes, c...)
	}

	if u, c := updateDesignDocFunctions(this.Filters, other.Filters, fmt.Sprintf("%v/_filters", this.Name)); u {

		updated = true
		changes = append(changes, c...)
	}

	if u, c := updateDesignDocFunctions(this.Updates, other.Updates, fmt.Sprintf("%v/_updates", this.Name)); u {

		updated = true
		changes = append(changes, c...)
	}

	if u, c := updateDesignDocFunctions(this.Validate, other.Validate, fmt.Sprintf("%v/_validate_on_update", this.Name)); u {

		updated = true
		changes = append(changes, c...)
	}

	return
}
