# logger
    这是一个基于GO语言的服务器日志系统，使用起来会非常方便，API接口简洁，易于嵌入到目前的项目工程中。


# 特性
    1.支持按日备份，跨天会创建新的日志
    2.支持按大小切分日志，如果单个日志文件超过指定上限，会重新创建日志
    3.支持控制台不同日志不同颜色显示，DEBUG和INFO日志默认输出白色，WARN输出黄色，ERROR输出红色
    4.支持捕获异常操作，并将异常信息及出错时运行堆栈保存在exception目录中，按时间存放
    
# 获取
    go get github.com/baickl/logger
    

# 示例

    import(
        "github.com/baickl/logger"
    )

    //初始化
    logger.Initialize("./log","LoginServer") 
      
    //设置选项 
    logger.SetConsole(true) 
    logger.SetLevel(logger.DEBUG)
      
    //单一输出 
    logger.Debug("I'm debug log!") 
    logger.Info("I'm info log!") 
    logger.Warn("I'm warn log!") 
    logger.Error("I'm error log!") 
      
    //格式化输出 
    logger.Debugf("I'm %s log! ","debug") 
    logger.Infof("I'm %s log!","info")
    logger.Warnf("I'm %s log!","warn")
    logger.Errorf("I'm %s log!","error")
      
    //行输出
    logger.Debugln("I'm","debug","log!") 
    logger.Infoln("I'm","info","log!")
    logger.Warnln("I'm","warn","log!") 
    logger.Errorln("I'm","error","log!")
    
    //异常捕获
    defer logger.CatchException()
    panic(err)  //此panic会被logger.CatchException()捕获，并保存到exception目录
