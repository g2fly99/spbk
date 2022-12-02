module spbk

go 1.15

require github.com/beego/beego/v2 v2.0.1

require (
	github.com/1lann/go-sip v0.0.0-20200718065607-c962f29a9181
	github.com/astaxie/beego v1.12.3
	github.com/chromedp/chromedp v0.7.6
	github.com/ghettovoice/gosip v0.0.0-20211221141116-292023b758f0 // indirect
	github.com/go-redis/redis v6.14.2+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/jart/gosip v0.0.0-20200629215808-4e7924e19438 // indirect
	github.com/smartystreets/goconvey v1.6.4
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/1lann/go-sip v0.0.0-20200718065607-c962f29a9181 => ../github.com/g2fly99/go-sip
