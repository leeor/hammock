package hammock

import (
	"fmt"
	"strings"

	"github.com/mikebell-org/go-couchdb"
)

type CouchDB struct {
	couchdb.CouchDB
}

func Database(host, database, username, password string) (*CouchDB, error) {

	db, err := couchdb.Database(host, database, username, password)

	return &CouchDB{*db}, err
}

func Sync(db *CouchDB, path string) error {

	// TODO: implement a document freezing option

	disk_data := newDesignDocCollection()
	fmt.Printf("%+v", disk_data)
	if err := disk_data.loadFromDisk(path); err == nil {

		db_data := newDesignDocument()

		for doc_name, document := range disk_data.Documents {

			if err := db.GetDocument(&db_data, fmt.Sprintf("%v", doc_name)); err == nil {

				if updated, changes := db_data.update(document); updated {

					fmt.Printf("DB code of %v needs to be updated:\n%v\n", doc_name, strings.Join(changes, "\n"))
					if success, err := db.PutDocument(db_data, doc_name); err != nil || !success.OK {

						return err
					}
				}
			}
		}
	} else {

		return err
	}

	return nil
}
