package model

import (
	"github.com/jinzhu/gorm"
	"os/exec"
	"fmt"
	"bufio"
	"strings"
	"io/ioutil"
	"crypto/md5"
	"github.com/tspn/wrk-load-testing-module/statics"
)

type Job struct{
	gorm.Model
	Name          	string
	TestError	int
	ExitInterrupt	bool
	Complete	bool
	RequestUrl    	string
	RequestMethod 	string
	Label         	string
	Testset       	uint
	Load          	string
	Script		string
	WrkResult	[]WrkResult
}

func (r *Job) KeyValueToLoad(keys, values []string){
	keyValue := map[string]string{}
	for i, key := range keys{
		keyValue[key] = values[i]
	}

	for key, value := range keyValue{
		if len(key) > 0 {
			r.Load += key + "=" + value + "&"
		}
	}
}



func (j *Job) RunWrk(ts Testcase, label, scriptFile string, db *gorm.DB)bool{
	t := ts.Thread
	c := ts.Connection
	d := ts.Duration

	url := j.RequestUrl
	var command *exec.Cmd

	command = exec.Command("wrk", "-t"+t, "-c"+c, "-d"+d, "-s", scriptFile,url)

	fmt.Println("label", label)
	if label == "time" {
		j.Label = j.Label + d + ","
	}else{
		j.Label = j.Label + c + ","
	}

	fmt.Println(command.Args)
	cmdReader, _ := command.StdoutPipe()
	scanner := bufio.NewScanner(cmdReader)
	var out string
	go func() {
		for scanner.Scan() {
			out = fmt.Sprintf("%s\n%s", out, scanner.Text())
			if strings.Contains(out, "Transfer"){
				break;
			}
		}
	}()
	command.Start()
	command.Wait()
	fmt.Println(out)

	wrk := WrkResult{}
	wrk.SetData(url, out)
	wrk.JobID = j.ID

	if wrk.IsError {
		j.TestError = j.TestError + 1
	}

	wrk.TestcaseID = ts.ID

	j.WrkResult = append(j.WrkResult, wrk)
	return wrk.IsError
}

func (j *Job) GenerateScript(filename string)string{
	script := ""
	script += fmt.Sprintf(statics.LUA_METHOD, j.RequestMethod)
	if len(j.Load) > 0{
		script += fmt.Sprintf(statics.LUA_LOAD, j.Load)
		script += fmt.Sprintf(statics.LUA_CONTENTTYPE, "application/x-www-form-urlencoded")
	}else if len(j.Script) > 0{
		script += j.Script
	}
	md5filename := md5.Sum([]byte(filename))
	fmt.Println(script)
	fullpath := fmt.Sprintf("lua/%x.lua", md5filename)
	err := ioutil.WriteFile(fullpath, []byte(script), 0644)
	fmt.Println(err)
	return fullpath
}
