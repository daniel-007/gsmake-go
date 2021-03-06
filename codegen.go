package gsmake

var codegen = `
{{define "main.go"}}
// generate builder for {{.TargetPath}}
package main
import "os"
import "fmt"
import "flag"
import "strings"
import "github.com/gsdocker/gslogger"
import "github.com/gsmake/gsmake"

var verbflag = flag.Bool("v", false, "print more debug information")
var context = gsmake.NewRunner("{{ospath .RootPath}}","{{ospath .TargetPath}}")
func main(){
    flag.Parse()
    gslogger.Console(gsmake.Logfmt, gsmake.LogTimefmt)
    if flag.NArg() < 1 {
        fmt.Println("expect task name")
        os.Exit(1)
    }
    if !*verbflag {
		gslogger.NewFlags(gslogger.ASSERT | gslogger.ERROR | gslogger.WARN | gslogger.INFO)
	}
    if err := context.Start(); err != nil {
        context.E("%s",err)
        gslogger.Join()
        os.Exit(1)
    }
    context.D("exec task [%s] with args : %s",flag.Arg(0),strings.Join(flag.Args()[1:]," "))
    if err := context.Run(flag.Arg(0),flag.Args()[1:]...); err != nil {
        context.E("%s",err)
        gslogger.Join()
        os.Exit(1)
    }
    gslogger.Join()
}
{{end}}
{{define "project.go"}}
package main
import "github.com/gsmake/gsmake"
import task "{{.Name}}/.gsmake"
func init(){
    {{range $key, $value := .Task}}
    context.Task(&gsmake.TaskCmd{
        Name : "{{$key}}",
        Description : "{{$value.Description}}",
        F : task.{{taskname $key}},
        Prev : {{prev $value.Prev}},
        Project : "{{$value.Package}}",
        Scope : "{{$value.Domain}}",
    })
    {{end}}
}
{{end}}
`
