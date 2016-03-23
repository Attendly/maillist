package maillist

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"gopkg.in/go-playground/validator.v8"

	"github.com/go-gorp/gorp"
	_ "github.com/go-sql-driver/mysql" // Overloading mysql for gorp
)

type table struct {
	name, selectStr string
}

type database struct {
	db     *sql.DB
	dbmap  *gorp.DbMap
	tables map[reflect.Type]table
}

type errNotFound struct{}

// ErrNotFound returned when an entity is not present in the database
var ErrNotFound = &errNotFound{}

var validate *validator.Validate

func (err *errNotFound) Error() string {
	return "Not found"
}

func openDatabase(address string) (d database, err error) {
	if address == "" {
		return d, errors.New("database address not set")
	}
	d.db, err = sql.Open("mysql", address+"?charset=utf8&parseTime=True")
	if err != nil {
		return
	}

	dialect := gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}
	d.dbmap = &gorp.DbMap{Db: d.db, Dialect: dialect}
	d.tables = make(map[reflect.Type]table)

	// d.addTable(models.MailList{}, "mail_list")
	if validate == nil {
		config := validator.Config{TagName: "validate"}
		validate = validator.New(&config)
	}
	return
}

func (d *database) insert(i interface{}) error {

	reflect.Indirect(reflect.ValueOf(i)).FieldByName("CreateTime").SetInt(time.Now().Unix())

	if err := validate.Struct(i); err != nil {
		return err
	}
	err := d.dbmap.Insert(i)
	return err
}

func (d *database) delete(i interface{}, id int64) error {
	t := reflect.TypeOf(i)
	table, ok := d.tables[t]
	if !ok {
		return fmt.Errorf("type %s not registered in db", t)
	}

	sql := fmt.Sprintf("update %s set status='deleted' where id=?", table.name)
	_, err := d.dbmap.Exec(sql, id)
	return err
}

func (d *database) update(i interface{}) error {
	if err := validate.Struct(i); err != nil {
		return err
	}
	_, err := d.dbmap.Update(i)
	return err
}

func (d *database) addTable(i interface{}, tableName string) (selectStatement string) {
	tablemap := d.dbmap.AddTableWithName(i, tableName)
	t := reflect.TypeOf(i)

	if _, hasID := t.FieldByName("ID"); hasID {
		tablemap.SetKeys(true, "ID")
	}

	var columns []string
	for _, c := range tablemap.Columns {
		if !c.Transient {
			columns = append(columns, tablemap.TableName+"."+c.ColumnName)
		}
	}

	selectStatement = fmt.Sprintf("%s\n", strings.Join(columns, ","))
	d.tables[t] = table{tableName, selectStatement}
	return selectStatement
}

func (d *database) selectString(i interface{}) string {
	t := reflect.TypeOf(i)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	table, ok := d.tables[t]
	if !ok {
		log.Fatalf("type %s not registered in db", t)
	}
	return table.selectStr
}
