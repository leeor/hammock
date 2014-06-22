# Hammock

Easily sync CouchDB design documents with an RCS of your choosing.

## Usage

Create the following directory structure:
```
root
`-- design_doc_name
    |-- views
    |   |-- view_name
    |   |   |-- map.js
    |   |   `-- reduce.js
    |   `-- another_view
    |       `-- map.js
    `-- [shows|lists|filters|updates|validate]
        |-- function_name1.js
        `-- function_name2.js
```
And call `Sync`:
```go

	if db, err := hammock.Database(db_host, db_name, db_username, db_password); err == nil {

		if changes, err := hammock.Sync(db, "root"); err == nil {

			if len(changes) > 0 {

				fmt.Println(strings.Join(changes, "\n"))
			}
		} else {

			log.Fatalln(err)
		}
	} else {

		log.Fatalln(err)
	}
```
`Sync` reads the directory structure and compares it with the design document in the CouchDB database (if such a design document does not exist it is created). Any differences are corrected treating the data read from disk as the correct version. Sync does not merge differences, it will always treat the state read from disk as the correct one.

The only difference `Sync` does not try to correct (yet) is deleting design documents that exist in the database but not on the disk.

## TODO
* Add tests
* Add an option for freezing documents, so that `Sync` will not attempt to override the CouchDB content.
* Consider deleting documents from CouchDB when they do not exist on disk.
