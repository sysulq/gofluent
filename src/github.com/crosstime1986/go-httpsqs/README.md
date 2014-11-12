go-httpsqs
==========

Twitter : [@ctpaul_com](https://twitter.com/ctpaul_com)

blog   : [http://www.ctpaul.com](http://www.ctpaul.com)

Httpsqs client which write by Golang


### 说明 

 - 服务端用 [HttpSqs](http://zyan.cc/httpsqs/) 有详细介绍，这里只针对Golang开发了轻量级的客户端
 
 - 单线程 Benchmark 效果如下, 分三组进行了单元测试和压力测试，第一组使用 `Get`、`Put` (非持久连接，且用自带的`net/http`包) ， 第二组使用 `PGets`、`PPuts` （已被第三次优化合并），由于服务端支持 Keep-alive, 所有持久连接，且用自带的`net/http`包，， 第三组简化http协议上的解释:
 
 ![Benchmark 效果](http://www.ctpaul.com/wp-content/uploads/2014/09/QQ%E6%88%AA%E5%9B%BE20140912134944.jpg "Benchmark 效果")
 
 - 支持 utf-8, gbk, 等国际化字符集
  
 
## 安装

 安装使用命令 `go get github.com/crosstime1986/go-httpsqs` 
 
 更新使用命令  `go get -u github.com/crosstime1986/go-httpsqs` 
 
## 使用
 
 使用非常简单 ，Here is the demo ：

```go
    package main
    
    import (
        "fmt"
        "time"
        httpsqs "github.com/crosstime1986/go-httpsqs"
    )
    
    func main() {
    
        q = httpsqs.NewClient("116.254.199.28", 1218, "", false)
        
        res, err  := q.Put("test", "你好，世界")        
        
        for {
            res, err  := q.Get("test")
            if err != nil {
                fmt.Println(err)
                time.Sleep(time.Second * 1)
            } else {
                fmt.Println(res.Data)
                fmt.Println(res.Pos)
            }
        }
    }
```

## 长链接协议 

`Gets` `Puts` 方法针 `keep-alive` 提供了长链接，分别是 `PGets` `PPuts`  (用于command-line模式下, 性能更好)

```go
    for i := 0; i < 5000; i++ {
        q.Put("test", "你好，世界")
    }
```