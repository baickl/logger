package logger

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type LEVEL int //日志等级
type COLOR int //显示颜色
type STYLE int //显示样式

const (
	ALL   LEVEL = iota //所有日志
	DEBUG              //调试
	INFO               //信息
	WARN               //警告
	ERROR              //错误
	FATAL              //崩溃
)

const (
	CLR_BLACK   = COLOR(30) // 黑色
	CLR_RED     = COLOR(31) // 红色
	CLR_GREEN   = COLOR(32) // 绿色
	CLR_YELLOW  = COLOR(33) // 黄色
	CLR_BLUE    = COLOR(34) // 蓝色
	CLR_PURPLE  = COLOR(35) // 紫红色
	CLR_CYAN    = COLOR(36) // 青蓝色
	CLR_WHITE   = COLOR(37) // 白色
	CLR_DEFAULT = COLOR(39) // 默认
)

const (
	STYLE_DEFAULT   = STYLE(0) //终端默认设置
	STYLE_HIGHLIGHT = STYLE(1) //高亮显示
	SYTLE_UNDERLINE = STYLE(4) //使用下划线
	SYTLE_BLINK     = STYLE(5) //闪烁
	STYLE_INVERSE   = STYLE(7) //反白显示
	STYLE_INVISIBLE = STYLE(8) //不可见
)

const (
	logFlags             = log.Ldate | log.Lmicroseconds | log.Lshortfile //日志输出flag
	logConsoleFlag       = 0                                              //console输出flag
	logDumpExceptionFlag = 0                                              //exception输出flag
	logMaxSize           = 512 * 1024 * 1024                              //单个日志文件最大大小
)

/******************************************************************************
 @brief
 	日志文件类结构
 @author
 	chenzhiguo
 @history
 	2015-05-16_10:22 	chenzhiguo		创建
*******************************************************************************/
type LOG_FILE struct {
	sync.RWMutex             //线程锁
	log_dir      string      //日志存放目录
	log_filename string      //日志基础名字
	timestamp    time.Time   //日志创建时的时间戳
	logfilepath  string      //当前日志路径
	logfile      *os.File    //当前日志文件实例
	logger       *log.Logger //当前日志操作实例
}

var (
	logLevel         LEVEL     = ALL  //日志级别
	logConsole       bool      = true //终端控制台显示控制，默认为true
	logConsolePrefix string           //终端控制台显示前缀
	logFile          *LOG_FILE        //日志文件实例
)

/******************************************************************************
 @brief
 	设置终端控制台是否显示日志
 @author
 	chenzhiguo
 @param
	isConsole			是否显示
 @return
 	-
 @history
 	2015-05-16_10:22 	chenzhiguo		创建
*******************************************************************************/
func SetConsole(isConsole bool) {
	logConsole = isConsole
}

/******************************************************************************
 @brief
 	设置终端控制台显示前缀
 @author
 	chenzhiguo
 @param
	prefix				要显示的前缀
 @return
 	-
 @history
 	2015-05-16_17:15 	chenzhiguo		创建
*******************************************************************************/
func SetConsolePrefix(prefix string) {
	logConsolePrefix = prefix
}

/******************************************************************************
 @brief
 	设置日志记录级别，低于这个级别的日志将会被丢弃
 @author
 	chenzhiguo
 @param
	_level				级别
 @return
 	-
 @history
 	2015-05-16_10:22 	chenzhiguo		创建
*******************************************************************************/
func SetLevel(_level LEVEL) {
	logLevel = _level
}

/******************************************************************************
 @brief
 	用颜色来显示字符串
 @author
 	chenzhiguo
 @param
	str					待显示的字符串
	s					显示样式
	fc					显示前景色
	bc					显示背景色
 @return
 	string				返回生成的格式化字符
 @history
 	2015-05-16_10:22 	chenzhiguo		创建
*******************************************************************************/
func SprintColor(str string, s STYLE, fc, bc COLOR) string {

	_fc := int(fc)      //前景色
	_bc := int(bc) + 10 //背景色

	return fmt.Sprintf("%c[%d;%d;%dm%s%c[0m", 0x1B, int(s), _bc, _fc, str, 0x1B)
}

/******************************************************************************
 @brief
 	日志系统初始化，需要提供存放目录和日志基础名字
 		例：
 			logger.Initialize("./log/","login_server")

 		那么日志系统会创建 ./log/20150516/login_server_1022.log 日志文件
 @author
 	chenzhiguo
 @param
	fileDir				日志存放路径
	fileName			日志基础名称
 @return
 	int					返回结果
 @history
 	2015-05-16_10:22 	chenzhiguo		创建
*******************************************************************************/
func Initialize(fileDir, fileName string) {

	//目录修正
	dir := fileDir
	if fileDir[len(fileDir)-1] == '\\' || fileDir[len(fileDir)-1] == '/' {
		dir = fileDir[:len(fileDir)-1]
	}

	//初始化结构体
	logFile = &LOG_FILE{log_dir: dir, log_filename: fileName, timestamp: time.Now()}
	logFile.Lock()
	defer logFile.Unlock()

	//创建文件
	fn := logFile.newlogfile()
	var err error
	logFile.logfile, err = os.OpenFile(fn, os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}

	//初始化日志
	logFile.logger = log.New(logFile.logfile, "", logFlags)
	log.SetFlags(logConsoleFlag)
	logFile.logfilepath = fn

	//启动文件监控模块
	go fileMonitor()
}

/******************************************************************************
 @brief
 	启动性能分析协程
 @author
 	chenzhiguo
 @param
	port				当前服务器端口，性能分析端口会监听端口:port+10000
 @return
 	-
 @history
 	2015-05-16_10:22 	chenzhiguo		创建
*******************************************************************************/
func StartPPROF(port int) {
	go func(_port int) {
		log.Println(http.ListenAndServe(fmt.Sprintf(":%d", 10000+_port), nil))
	}(port)
}

/******************************************************************************
 @brief
 	捕获异常处理函数，如果发现程序有异常以后，会在当前目录中创建异常目录，并将异常信息写入到文件中
 		例如：
 			func foo(){
				defer logger.CatchException()
				panic(123)	//此panic会被logger.CatchException()函数捕获并记录
 			}
 @author
 	chenzhiguo
 @param
	-
 @return
 	-
 @history
 	2015-05-16_10:22 	chenzhiguo		创建
*******************************************************************************/
func CatchException() {

	if err := recover(); err != nil {
		logfile, err2 := os.OpenFile(newDumpFile(), os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err2 != nil {
			return
		}

		defer logfile.Close()
		logger := log.New(logfile, "", logFlags)
		logger.SetFlags(logDumpExceptionFlag)

		strLog := fmt.Sprintf(`
===============================================================================
EXCEPTION: %#v																			
===============================================================================		
%s`,
			err,
			string(debug.Stack()))

		logger.Println(strLog)
		fmt.Println(strLog)
	}
}

/******************************************************************************
 @brief
 	生成新的dump文件路径
 @author
 	chenzhiguo
 @param
	-
 @return
 	string				返回dump文件路径
 @history
 	2015-05-16_10:22 	chenzhiguo		创建
*******************************************************************************/
func newDumpFile() string {

	now := time.Now()

	filename := fmt.Sprintf("exceptions.%02d_%02d_%02d", now.Hour(), now.Minute(), now.Second())
	dir := fmt.Sprintf("./exceptions/%04d-%02d-%02d/", now.Year(), int(now.Month()), now.Day())
	os.MkdirAll(dir, os.ModePerm)
	fn := fmt.Sprintf("%s%s.log", dir, filename)
	if !isFileExist(fn) {
		return fn
	}

	n := 1
	for {
		fn = fmt.Sprintf("%s%s_%d.log", dir, filename, n)
		if !isFileExist(fn) {
			break
		}

		n += 1
	}

	return fn

}

type Verbose bool

func V(level LEVEL) Verbose {

	if logLevel >= level {
		return Verbose(true)
	}

	return Verbose(false)
}

func (v Verbose) Debug(arg interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("DEBUG %s", fmt.Sprintln(arg))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(DEBUG, context)

}

func (v Verbose) Info(arg interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("INFO %s", fmt.Sprintln(arg))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(INFO, context)

}

func (v Verbose) Warn(arg interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("WARN %s", fmt.Sprintln(arg))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(WARN, context)

}

func (v Verbose) Error(arg interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("ERROR %s", fmt.Sprintln(arg))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(ERROR, context)
}

func (v Verbose) Fatal(arg interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("FATAL %s", fmt.Sprintln(arg))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(FATAL, context)
}

func (v Verbose) Debugf(format string, args ...interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("DEBUG %s", fmt.Sprintf(format, args...))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(DEBUG, context)
}

func (v Verbose) Infof(format string, args ...interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("INFO %s", fmt.Sprintf(format, args...))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(INFO, context)
}

func (v Verbose) Warnf(format string, args ...interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("WARN %s", fmt.Sprintf(format, args...))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(WARN, context)
}

func (v Verbose) Errorf(format string, args ...interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("ERROR %s", fmt.Sprintf(format, args...))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(ERROR, context)
}

func (v Verbose) Fatalf(format string, args ...interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("FATAL %s", fmt.Sprintf(format, args...))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(FATAL, context)
}

func (v Verbose) Debugln(args ...interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("DEBUG %s", fmt.Sprintln(args...))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(DEBUG, context)
}

func (v Verbose) Infoln(args ...interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("INFO %s", fmt.Sprintln(args...))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(INFO, context)
}

func (v Verbose) Warnln(args ...interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("WARN %s", fmt.Sprintln(args...))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(WARN, context)
}

func (v Verbose) Errorln(args ...interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("ERROR %s", fmt.Sprintln(args...))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(ERROR, context)
}

func (v Verbose) Fatalln(args ...interface{}) {
	if !v {
		return
	}

	defer catchError()

	context := fmt.Sprintf("FATAL %s", fmt.Sprintln(args...))
	context = strings.TrimRight(context, "\n")
	if logFile != nil {
		logFile.RLock()
		defer logFile.RUnlock()
		logFile.logger.Output(2, context)
	}
	console(FATAL, context)
}

/******************************************************************************
 @brief
 	输出Debug日志
 @author
 	chenzhiguo
 @param
	arg					要输出的内容
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Debug(arg interface{}) {
	defer catchError()
	if logLevel <= DEBUG {
		context := fmt.Sprintf("DEBUG %s", fmt.Sprintln(arg))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(DEBUG, context)
	}
}

/******************************************************************************
 @brief
 	输出Info日志
 @author
 	chenzhiguo
 @param
	arg					要输出的内容
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Info(arg interface{}) {
	defer catchError()
	if logLevel <= INFO {
		context := fmt.Sprintf("INFO %s", fmt.Sprintln(arg))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(INFO, context)
	}
}

/******************************************************************************
 @brief
 	输出Warn日志
 @author
 	chenzhiguo
 @param
	arg					要输出的内容
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Warn(arg interface{}) {
	defer catchError()
	if logLevel <= WARN {
		context := fmt.Sprintf("WARN %s", fmt.Sprintln(arg))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(WARN, context)
	}
}

/******************************************************************************
 @brief
 	输出Error日志
 @author
 	chenzhiguo
 @param
	arg					要输出的内容
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Error(arg interface{}) {
	defer catchError()
	if logLevel <= ERROR {
		context := fmt.Sprintf("ERROR %s", fmt.Sprintln(arg))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(ERROR, context)
	}
}

/******************************************************************************
 @brief
 	输出Fatal日志
 @author
 	chenzhiguo
 @param
	arg					要输出的内容
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Fatal(arg interface{}) {
	defer catchError()
	if logLevel <= FATAL {
		context := fmt.Sprintf("FATAL %s", fmt.Sprintln(arg))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(FATAL, context)
	}
}

/******************************************************************************
 @brief
 	输出Debug日志，支持格式化操作
 @author
 	chenzhiguo
 @param
	format				格式化字符或字符串
	args				格式化参数
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Debugf(format string, args ...interface{}) {
	defer catchError()
	if logLevel <= DEBUG {
		context := fmt.Sprintf("DEBUG %s", fmt.Sprintf(format, args...))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(DEBUG, context)
	}
}

/******************************************************************************
 @brief
 	输出Info日志，支持格式化操作
 @author
 	chenzhiguo
 @param
	format				格式化字符或字符串
	args				格式化参数
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Infof(format string, args ...interface{}) {
	defer catchError()
	if logLevel <= INFO {
		context := fmt.Sprintf("INFO %s", fmt.Sprintf(format, args...))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(INFO, context)
	}
}

/******************************************************************************
 @brief
 	输出Warn日志，支持格式化操作
 @author
 	chenzhiguo
 @param
	format				格式化字符或字符串
	args				格式化参数
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Warnf(format string, args ...interface{}) {
	defer catchError()
	if logLevel <= WARN {
		context := fmt.Sprintf("WARN %s", fmt.Sprintf(format, args...))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(WARN, context)
	}
}

/******************************************************************************
 @brief
 	输出Error日志，支持格式化操作
 @author
 	chenzhiguo
 @param
	format				格式化字符或字符串
	args				格式化参数
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Errorf(format string, args ...interface{}) {
	defer catchError()
	if logLevel <= ERROR {
		context := fmt.Sprintf("ERROR %s", fmt.Sprintf(format, args...))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(ERROR, context)
	}
}

/******************************************************************************
 @brief
 	输出Fatal日志，支持格式化操作
 @author
 	chenzhiguo
 @param
	format				格式化字符或字符串
	args				格式化参数
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Fatalf(format string, args ...interface{}) {
	defer catchError()
	if logLevel <= FATAL {
		context := fmt.Sprintf("FATAL %s", fmt.Sprintf(format, args...))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(FATAL, context)
	}
}

/******************************************************************************
 @brief
 	输出Debug日志
 @author
 	chenzhiguo
 @param
	format				格式化字符或字符串
	args				格式化参数
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Debugln(args ...interface{}) {
	defer catchError()
	if logLevel <= DEBUG {
		context := fmt.Sprintf("DEBUG %s", fmt.Sprintln(args...))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(DEBUG, context)
	}
}

/******************************************************************************
 @brief
 	输出Info日志
 @author
 	chenzhiguo
 @param
	format				格式化字符或字符串
	args				格式化参数
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Infoln(args ...interface{}) {
	defer catchError()
	if logLevel <= INFO {
		context := fmt.Sprintf("INFO %s", fmt.Sprintln(args...))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(INFO, context)
	}
}

/******************************************************************************
 @brief
 	输出Warn日志
 @author
 	chenzhiguo
 @param
	format				格式化字符或字符串
	args				格式化参数
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Warnln(args ...interface{}) {
	defer catchError()
	if logLevel <= WARN {
		context := fmt.Sprintf("WARN %s", fmt.Sprintln(args...))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(WARN, context)
	}
}

/******************************************************************************
 @brief
 	输出Error日志
 @author
 	chenzhiguo
 @param
	format				格式化字符或字符串
	args				格式化参数
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Errorln(args ...interface{}) {
	defer catchError()
	if logLevel <= ERROR {
		context := fmt.Sprintf("ERROR %s", fmt.Sprintln(args...))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(ERROR, context)
	}
}

/******************************************************************************
 @brief
 	输出Fatal日志
 @author
 	chenzhiguo
 @param
	format				格式化字符或字符串
	args				格式化参数
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func Fatalln(args ...interface{}) {
	defer catchError()
	if logLevel <= FATAL {
		context := fmt.Sprintf("FATAL %s", fmt.Sprintln(args...))
		context = strings.TrimRight(context, "\n")
		if logFile != nil {
			logFile.RLock()
			defer logFile.RUnlock()
			logFile.logger.Output(2, context)
		}
		console(FATAL, context)
	}
}

/******************************************************************************
 @brief
 	输出检查日志是否需要重新命名，比如说跨天，大小变化，或是文件已经不存在
 @author
 	chenzhiguo
 @param
	-
 @return
 	bool				返回true表示需要重命名，否则不需要
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func (f *LOG_FILE) isMustRename() bool {

	//检查是否跨天
	if f.checkFileDate() {
		return true
	}

	//检查大小是否有变化
	if f.checkFileSize() {
		return true
	}

	//检查文件是否存在
	if f.checkFileExist() {
		return true
	} else {
		os.MkdirAll(f.log_dir, os.ModePerm)
	}

	return false
}

/******************************************************************************
 @brief
 	检查文件日期是否已经跨天
 @author
 	chenzhiguo
 @param
	-
 @return
 	bool				返回true表示已经跨天，否则没有
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func (f *LOG_FILE) checkFileDate() bool {
	if time.Now().YearDay() != f.timestamp.YearDay() {
		return true
	}

	return false
}

/******************************************************************************
 @brief
 	检查文件大小是否已经超过指定的大小
 @author
 	chenzhiguo
 @param
	-
 @return
 	bool				返回true表示已经超过，否则没有
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func (f *LOG_FILE) checkFileSize() bool {

	fileInfo, err := os.Stat(f.logfilepath)
	if err != nil {
		return false
	}

	if fileInfo.Size() >= logMaxSize {
		return true
	}

	return false
}

/******************************************************************************
 @brief
 	检查文件大小是否存在
 @author
 	chenzhiguo
 @param
	-
 @return
 	bool				返回true表示存在，否则表示没有有文件或是没有权限访问
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func (f *LOG_FILE) checkFileExist() bool {
	if !isFileExist(f.logfilepath) {
		return true
	}

	return false
}

/******************************************************************************
 @brief
 	执行文件重命名操作
 @author
 	chenzhiguo
 @param
	-
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func (f *LOG_FILE) rename() {
	f.timestamp = time.Now()
	fn := f.newlogfile()

	if f.logfile != nil {
		f.logfile.Close()
	}

	f.logfile, _ = os.OpenFile(fn, os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModePerm)
	f.logger = log.New(logFile.logfile, "", logFlags)
	f.logfilepath = fn
}

/******************************************************************************
 @brief
 	获取新的日志文件的名称
 @author
 	chenzhiguo
 @param
	-
 @return
 	string				返回名称
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func (f *LOG_FILE) newlogfile() string {

	dir := fmt.Sprintf("%s/%04d-%02d-%02d/", f.log_dir, f.timestamp.Year(), f.timestamp.Month(), f.timestamp.Day())
	os.MkdirAll(dir, os.ModePerm)
	filename := fmt.Sprintf("%s/%s.%02d_%02d_%02d", dir, f.log_filename, f.timestamp.Hour(), f.timestamp.Minute(), f.timestamp.Second())

	fn := filename + ".log"
	if !isFileExist(fn) {
		return fn
	}

	n := 1
	for {
		fn = fmt.Sprintf("%s_%d.log", filename, n)
		if !isFileExist(fn) {
			break
		}
		n += 1
	}

	return fn
}

/******************************************************************************
 @brief
 	判断文件是否存在
 @author
 	chenzhiguo
 @param
	file				文件路径
 @return
 	bool				返回true表示文件存在，否则表示不存在
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func isFileExist(file string) bool {
	finfo, err := os.Stat(file)
	if err != nil {
		return false
	}
	if finfo.IsDir() {
		return false
	} else {
		return true
	}
}

/******************************************************************************
 @brief
 	输出信息到终端控制台上
 @author
 	chenzhiguo
 @param
	ll					日志等级
	args				要输出的内容
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func console(ll LEVEL, args string) {
	if logConsole {
		_, file, line, _ := runtime.Caller(2)
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short

		now := time.Now()

		context := ""
		if len(logConsolePrefix) > 0 {
			context = fmt.Sprintf("==>[%04d/%02d/%02d_%02d:%02d:%02d.%06d] @%s #%s:%d %s", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), time.Duration(now.Nanosecond())/(time.Microsecond), logConsolePrefix, file, line, args)
		} else {
			context = fmt.Sprintf("==>[%04d/%02d/%02d_%02d:%02d:%02d.%06d] #%s:%d %s", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), time.Duration(now.Nanosecond())/(time.Microsecond), file, line, args)
		}

		switch ll {
		case DEBUG:
			log.Println(SprintColor(context, STYLE_DEFAULT, CLR_DEFAULT, CLR_DEFAULT))
		case INFO:
			log.Println(SprintColor(context, STYLE_DEFAULT, CLR_DEFAULT, CLR_DEFAULT))
		case WARN:
			log.Println(SprintColor(context, STYLE_DEFAULT, CLR_YELLOW, CLR_DEFAULT))
		case ERROR:
			log.Println(SprintColor(context, STYLE_HIGHLIGHT, CLR_RED, CLR_DEFAULT))
		case FATAL:
			log.Println(SprintColor(context, STYLE_HIGHLIGHT, CLR_PURPLE, CLR_DEFAULT))
		default:
			log.Println(SprintColor(context, STYLE_DEFAULT, CLR_DEFAULT, CLR_DEFAULT))
		}

	}
}

/******************************************************************************
 @brief
 	捕获程序错误，仅供内部使用
 @author
 	chenzhiguo
 @param
	-
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func catchError() {
	if err := recover(); err != nil {
		log.Println("err", err)
	}
}

/******************************************************************************
 @brief
 	文件监控函数，循环检测文件是否需要重命名
 @author
 	chenzhiguo
 @param
	-
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func fileMonitor() {
	timer := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-timer.C:
			fileCheck()
		}
	}
}

/******************************************************************************
 @brief
 	检查文件是否需要重命名，如果需要，那么执行重命名逻辑
 @author
 	chenzhiguo
 @param
	-
 @return
 	-
 @history
 	2015-05-16_10:52 	chenzhiguo		创建
*******************************************************************************/
func fileCheck() {

	defer catchError()
	if logFile != nil && logFile.isMustRename() {
		logFile.Lock()
		defer logFile.Unlock()
		logFile.rename()
	}
}
