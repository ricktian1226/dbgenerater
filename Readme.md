#数据库生成工具
## 配置
	"表名" : {
       "comment" : "表注释"
       "列名1" ： {
             "comment" : "列注释",
             "pk" : true,
             "null" : false,
             "type" : 数据类型,
             "size" : 长度,
             "default" ： 默认值,
		},
       "列名2" ： {
			 "comment" : "列注释",
             "pk" : false,
             "null" : true,
             "type" : 数据类型,
             "size" : 长度,
             "default" ： 默认值,
		},
        "index" : [
            ["列名", "列名"],
            ["列名", "列名", "列名"]
        ]，
	}
## model 生成
models下的代码通过该工具自动生成
## sql 生成
### create sql
创建表的数据库脚本<br>
文件路径：marco_uc_server\sql\__all_table_create.sql
### alter sql
修改表的数据库脚本<br>
文件路径：marco_uc_server\sql\__all_table_alter.sql
### drop sql
删除表字段的数据库脚本<br>
文件路径：marco_uc_server\sql\__all_table_drop.sql
### add sql
增加表字段的数据库脚本<br>
文件路径：marco_uc_server\sql\__all_table_add.sql
## mysqldump 脚本生成

