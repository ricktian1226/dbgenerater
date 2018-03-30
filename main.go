package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
)

const (
	CONFIG_FILE = "config.json"

	MODEL_DIR = "../../models/"
	SQL_DIR   = "../../sql/"

	COMMENT    = "comment"
	PK         = "pk"
	NULL       = "null"
	TYPE       = "type"
	DEFAULT    = "default"
	MODEL_NAME = "model_name"
	INDEX      = "index"
	SN         = "sn"
	SIZE       = "size"
)

//数据库数据类型与model数据类型的映射
var DATA2MODEL_TYPE_MAP = map[string]string{
	"tinyint unsigned":  "uint8",
	"tinyint":           "int8",
	"smallint unsigned": "uint16",
	"smallint":          "int16",
	"int unsigned":      "uint32",
	"int":               "int32",
	"bigint unsigned":   "uint64",
	"bigint":            "int64",
	"char":              "string",
	"varchar":           "string",
}

//表定义
type table struct {
	name      string    //表名
	comment   string    //表注释
	columns   []*column //列列表
	indexs    []*index  //索引定义
	modelName string
}

func NewTable(name string) *table {
	return &table{
		name: name,
	}
}

//列
type column struct {
	name         string
	comment      string
	sType        string
	iSize        int
	iDefault     interface{}
	isPk, isNull bool
	modelName    string
	sn           int
}

func NewColumn() *column {
	return &column{}
}

func (c *column) parse(columnMap map[string]interface{}) (err error) {
	for k, v := range columnMap {
		switch k {
		case TYPE:
			{
				if sType, ok := v.(string); ok {
					if _, ok := DATA2MODEL_TYPE_MAP[sType]; ok {
						c.sType = sType
					} else {
						log.Fatalf("不支持的TYPE类型 %s", sType)
						return ERR_PARSE_FAILED
					}
				} else {
					log.Fatalf("解析 TYPE %v 失败", v)
					return ERR_PARSE_FAILED
				}
			}
		case PK:
			{
				if isPk, ok := v.(bool); ok {
					c.isPk = isPk
				} else {
					log.Fatalf("解析 PK %v 失败", v)
					return ERR_PARSE_FAILED
				}
			}
		case NULL:
			{
				if isNull, ok := v.(bool); ok {
					c.isNull = isNull
				} else {
					log.Fatalf("解析 NULL %v 失败", v)
					return ERR_PARSE_FAILED
				}
			}
		case DEFAULT:
			{
				c.iDefault = v
			}
		case MODEL_NAME:
			{
				if modelName, ok := v.(string); ok {
					c.modelName = modelName
				} else {
					log.Fatalf("解析 MODEL_NAME %v 失败", v)
					return ERR_PARSE_FAILED
				}
			}
		case COMMENT:
			{
				if comment, ok := v.(string); ok {
					c.comment = comment
				} else {
					log.Fatalf("解析 COMMENT %v 失败", v)
					return ERR_PARSE_FAILED
				}
			}
		case SN:
			{
				//c.sn = v.(int)
				if sn, ok := v.(float64); ok {
					c.sn = int(sn)
				} else {
					log.Fatalf("解析 SN %v 失败", v)
					return ERR_PARSE_FAILED
				}
			}
		case SIZE:
			{
				if size, ok := v.(float64); ok {
					c.iSize = int(size)
				} else {
					log.Fatalf("解析 SIZE %v 失败", v)
					return ERR_PARSE_FAILED
				}
			}
		}
	}

	return
}

//索引
type index struct {
	columns []string
}

func NewIndex() *index {
	return &index{}
}

type TableSlice []*table

func (t *table) parseIndex(indexSlice []interface{}) (err error) {
	for _, indexs := range indexSlice {
		idx := NewIndex()
		if indexStrs, ok := indexs.([]interface{}); ok {
			for _, indexStr := range indexStrs {
				if s, ok := indexStr.(string); ok {
					idx.columns = append(idx.columns, s)
				} else {
					log.Fatalf("解析 INDEX %v 失败", indexStr)
					return ERR_PARSE_FAILED
				}
			}
		} else {
			log.Fatalf("解析 INDEX %v 失败", indexStrs)
			return ERR_PARSE_FAILED
		}

		t.indexs = append(t.indexs, idx)
	}

	return
}

//生成model文件
func (t *table) outputModelCode() (err error) {
	modelCode := fmt.Sprintf(""+
		"package models\n\n"+
		"type %s struct{\n", t.modelName)
	isExistPk := false
	for _, c := range t.columns {
		modelCode += fmt.Sprintf("\t%s %s `orm:\"column(%s)", c.modelName, DATA2MODEL_TYPE_MAP[c.sType], c.name)
		if c.isNull {
			modelCode += ";null"
		}

		if c.isPk && !isExistPk { //orm只支持一个pk，原因是什么？
			modelCode += ";pk"
			isExistPk = true
		}

		if c.iSize > 0 {
			modelCode += fmt.Sprintf(";size(%d)", c.iSize)
		}

		modelCode += "\"`\n"
	}

	modelCode += "}\n"

	err = ioutil.WriteFile(MODEL_DIR+t.name+".go", []byte(modelCode), os.FileMode(0666))
	if err != nil {
		log.Fatalf("写入文件 %s 失败 : %v", MODEL_DIR+t.name+".go", err)
		return
	}

	return
}

func outputSql(createSql, alterSql, dropSql, addSql string) (err error) {
	file := []string{
		"__all_table_create.sql", "__all_table_field_alter.sql", "__all_table_field_drop.sql", "__all_table_field_add.sql",
	}
	sql := []string{
		createSql, alterSql, dropSql, addSql,
	}
	for i := 0; i < len(file); i++ {
		err = ioutil.WriteFile(SQL_DIR+file[i], []byte(sql[i]), os.FileMode(0666))
		if err != nil {
			log.Fatalf("写入文件 %s 失败 : %v", SQL_DIR+file[i], err)
			return
		}
	}

	return
}

func (t *table) generateSQL() (err error, createSql, alterSql, dropSql string) {
	//create sql
	err, createSql = t.generateCreateSQL()
	if err != nil {
		return
	}

	//alter sql
	err, alterSql = t.generateAlterSQL()
	if err != nil {
		return
	}

	//drop sql
	err, dropSql = t.generateDropSQL()
	if err != nil {
		return
	}

	return
}

func (t *table) generateCreateSQL() (err error, sql string) {
	sql += "-- --------------------------------------------------\n"
	sql += fmt.Sprintf("--  Table Structure for `models.%s`\n", t.modelName)
	sql += "-- --------------------------------------------------\n"
	sql += fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n", t.name)
	var primaryKeyStr string
	for i, c := range t.columns {
		if c.sType == "varchar" || c.sType == "char" {
			sql += fmt.Sprintf("`%s` %s(%d)", c.name, c.sType, c.iSize)
		} else {
			sql += fmt.Sprintf("`%s` %s", c.name, c.sType)
		}

		if c.isNull {
			sql += " NULL"
		} else {
			sql += " NOT NULL"
		}

		if c.isPk {
			if len(primaryKeyStr) == 0 {
				primaryKeyStr = fmt.Sprintf("`%s`", c.name)
			} else {
				primaryKeyStr += fmt.Sprintf(",`%s`", c.name)
			}
		}

		if c.iDefault != nil {
			if c.sType == "varchar" || c.sType == "char" {
				sql += " DEFAULT ''"
			} else {
				sql += fmt.Sprintf(" DEFAULT %v", c.iDefault)
			}

		}

		if i == len(t.columns)-1 && len(primaryKeyStr) == 0 {
			sql += fmt.Sprintf(" COMMENT '%s'\n", c.comment)
		} else {
			sql += fmt.Sprintf(" COMMENT '%s',\n", c.comment)
		}

	}

	if len(primaryKeyStr) > 0 {
		sql += fmt.Sprintf("PRIMARY KEY(%s)\n", primaryKeyStr)
	}

	sql += fmt.Sprintf(") ENGINE=InnoDB COMMENT='%s' DEFAULT CHARSET=utf8;\n", t.comment)
	if len(t.indexs) > 0 {
		for _, index := range t.indexs {
			indexName := t.name
			var indexContent string
			for i, c := range index.columns {
				indexName += "_" + c
				if i == 0 {
					indexContent += fmt.Sprintf("`%s`", c)
				} else {
					indexContent += fmt.Sprintf(", `%s`", c)
				}
			}
			sql += fmt.Sprintf("CREATE INDEX `%s` ON `%s` (%s);\n", indexName, t.name, indexContent)
		}
	}

	sql += "\n"

	return
}

func (t *table) generateAlterSQL() (err error, sql string) {
	sql += "----------------------------------------------------\n"
	sql += fmt.Sprintf("--  `%s`\n", t.name)
	sql += "----------------------------------------------------\n"

	for _, c := range t.columns {
		sql += fmt.Sprintf("ALTER TABLE `%s` CHANGE `%s` ", t.name, c.name)
		if c.sType == "varchar" || c.sType == "char" {
			sql += fmt.Sprintf("`%s` %s(%d)", c.name, c.sType, c.iSize)
		} else {
			sql += fmt.Sprintf("`%s` %s", c.name, c.sType)
		}

		if c.isNull {
			sql += " NULL"
		} else {
			sql += " NOT NULL"
		}

		if c.iDefault != nil {
			if c.sType == "varchar" || c.sType == "char" {
				sql += " DEFAULT ''"
			} else {
				sql += fmt.Sprintf(" DEFAULT %v", c.iDefault)
			}

		}

		sql += fmt.Sprintf(" COMMENT '%s';\n", c.comment)
	}
	sql += "\n"

	return
}

func (t *table) generateDropSQL() (err error, sql string) {
	sql += "----------------------------------------------------\n"
	sql += fmt.Sprintf("--  `%s`\n", t.name)
	sql += "----------------------------------------------------\n"

	for _, c := range t.columns {
		sql += fmt.Sprintf("ALTER TABLE `%s` DROP `%s`;\n", t.name, c.name)
	}

	sql += "\n"

	return
}

func (t *table) generateAddSQL() (err error, sql string) {

	var preCName string
	sql += "----------------------------------------------------\n"
	sql += fmt.Sprintf("--  `%s`\n", t.name)
	sql += "----------------------------------------------------\n"

	for _, c := range t.columns {
		sql += fmt.Sprintf("ALTER TABLE `%s` ADD ", t.name)
		if c.sType == "varchar" || c.sType == "char" {
			sql += fmt.Sprintf("`%s` %s(%d)", c.name, c.sType, c.iSize)
		} else {
			sql += fmt.Sprintf("`%s` %s", c.name, c.sType)
		}

		if c.isNull {
			sql += " NULL"
		} else {
			sql += " NOT NULL"
		}

		if c.iDefault != nil {
			if c.sType == "varchar" || c.sType == "char" {
				sql += " DEFAULT ''"
			} else {
				sql += fmt.Sprintf(" DEFAULT %v", c.iDefault)
			}

		}

		if len(preCName) > 0 {
			sql += fmt.Sprintf(" COMMENT '%s' AFTER `%s`;\n", c.comment, preCName)
		} else {
			sql += fmt.Sprintf(" COMMENT '%s';\n", c.comment)
		}

		preCName = c.name
	}
	sql += "\n"

	return
}

var ERR_PARSE_FAILED = errors.New("解析失败")

func main() {
	var content map[string]interface{}
	if data, err := ioutil.ReadFile(CONFIG_FILE); err != nil {
		log.Fatalf("读取文件 %s 失败，%v", CONFIG_FILE, err)
		return
	} else {
		if err = json.Unmarshal(data, &content); err != nil {
			log.Fatalf("解码内容失败 %v", err)
			return
		} else {
			var tableSlice TableSlice
			for k, v := range content {
				log.Printf("%v : %v\n", k, v)
				maps, ok := v.(map[string]interface{})
				if !ok {
					log.Fatalf("转换失败\n")
					return
				}

				var (
					t = NewTable(k)
				)

				for k1, v1 := range maps {
					switch k1 {
					case MODEL_NAME:
						{
							if t.modelName, ok = v1.(string); !ok {
								log.Fatalf("解析 COMMENT %v 失败", v)
								return
							}
						}
					case COMMENT:
						{
							if t.comment, ok = v1.(string); !ok {
								log.Fatalf("解析 COMMENT %v 失败", v)
								return
							}
						}

					case INDEX:
						{
							err = t.parseIndex(v1.([]interface{}))
							if err != nil {
								return
							}
						}

					default:
						m, ok := v1.(map[string]interface{})
						if !ok {
							log.Fatalf("转换失败\n")
							return
						}
						c := NewColumn()
						if err = c.parse(m); err != nil {
							log.Fatalf("转换失败 : %v\n", err)
							return
						} else {
							c.name = k1
							t.columns = append(t.columns, c)
						}
					}
				}

				if t != nil {
					sort.Slice(t.columns, func(i, j int) bool {
						return t.columns[i].sn < t.columns[j].sn
					})
					tableSlice = append(tableSlice, t)
				}
			}

			err = outputModelCommonCode(tableSlice)
			if err != nil {
				return
			}

			//导出model代码源文件
			for _, t := range tableSlice {
				t.outputModelCode()
			}

			//导出sql
			var (
				createSql, alterSql, dropSql, addSql string
			)

			for _, t := range tableSlice {
				var sql string
				err, sql = t.generateCreateSQL()
				if err != nil {
					return
				}
				createSql += sql

				sql = ""
				err, sql = t.generateAlterSQL()
				if err != nil {
					return
				}
				alterSql += sql

				sql = ""
				err, sql = t.generateDropSQL()
				if err != nil {
					return
				}
				dropSql += sql

				sql = ""
				err, sql = t.generateAddSQL()
				if err != nil {
					return
				}
				addSql += sql
			}

			outputSql(createSql, alterSql, dropSql, addSql)
		}
	}
}

//生成model文件
func outputModelCommonCode(tables TableSlice) (err error) {
	code := "package models\n\n" +
		"import \"marco_uc_server/common\"\n\n" +
		"func MODELS_INIT() (err error) {\n" +
		"	common.DB_REGISTER_MODELS(\n"
	for _, t := range tables {
		code += "\t\tnew(" + t.modelName + "),\n"
	}
	code += ")\n" +
		"\nreturn\n" +
		"}\n"

	err = ioutil.WriteFile(MODEL_DIR+"common.go", []byte(code), os.FileMode(0666))
	if err != nil {
		log.Fatalf("写入文件 %s 失败 : %v", MODEL_DIR+"common.go", err)
		return
	}

	return
}
