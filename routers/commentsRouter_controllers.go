package routers

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context/param"
)

func init() {

    beego.GlobalControllerRouter["spbk/controllers:BlackController"] = append(beego.GlobalControllerRouter["spbk/controllers:BlackController"],
        beego.ControllerComments{
            Method: "Add",
            Router: "/add",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["spbk/controllers:BlackController"] = append(beego.GlobalControllerRouter["spbk/controllers:BlackController"],
        beego.ControllerComments{
            Method: "Config",
            Router: "/config",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["spbk/controllers:BlackController"] = append(beego.GlobalControllerRouter["spbk/controllers:BlackController"],
        beego.ControllerComments{
            Method: "Delete",
            Router: "/delete",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["spbk/controllers:BlackController"] = append(beego.GlobalControllerRouter["spbk/controllers:BlackController"],
        beego.ControllerComments{
            Method: "DetectBlack",
            Router: "/detectblack",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["spbk/controllers:BlackController"] = append(beego.GlobalControllerRouter["spbk/controllers:BlackController"],
        beego.ControllerComments{
            Method: "SmsVerify",
            Router: "/smsverify",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["spbk/controllers:BlackController"] = append(beego.GlobalControllerRouter["spbk/controllers:BlackController"],
        beego.ControllerComments{
            Method: "Statics",
            Router: "/statics",
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["spbk/controllers:BlackController"] = append(beego.GlobalControllerRouter["spbk/controllers:BlackController"],
        beego.ControllerComments{
            Method: "SmsVerifys",
            Router: "/verify",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["spbk/controllers:BlackController"] = append(beego.GlobalControllerRouter["spbk/controllers:BlackController"],
        beego.ControllerComments{
            Method: "Verify",
            Router: "/verifyo",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["spbk/controllers:BlackController"] = append(beego.GlobalControllerRouter["spbk/controllers:BlackController"],
        beego.ControllerComments{
            Method: "Verifys",
            Router: "/verifys",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["spbk/controllers:M5gController"] = append(beego.GlobalControllerRouter["spbk/controllers:M5gController"],
        beego.ControllerComments{
            Method: "GetIp",
            Router: "/ip",
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["spbk/controllers:M5gController"] = append(beego.GlobalControllerRouter["spbk/controllers:M5gController"],
        beego.ControllerComments{
            Method: "Option",
            Router: "/option",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["spbk/controllers:M5gController"] = append(beego.GlobalControllerRouter["spbk/controllers:M5gController"],
        beego.ControllerComments{
            Method: "Send",
            Router: "/send",
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

}
