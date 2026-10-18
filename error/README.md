# MyError
## 说明
自定义Error类型的管理  
## 初衷
go原生的error只能传递一个string，这导致：error的等级，发生error时的HTTP码，发生error时应该向用户展示什么都需要在打印日志、或者向用户返还数据的时候进行判断。这样做，我认为会丢失发生这个error时的上下文，使得结果可能和error发生时的初衷有出入，所以我设计了这个自定义error，httpMessage，httpCode，logLevel在错误发生阶段就可以确定，在取用的时候只需要转化为*newerror.Error即可提取。
## 功能
### MakeError
用来生成最常规的，包含了httpMessage,httpCode,LogLevel,errorCode的基础*newerror.Error
### TranslateError
考虑到需要频繁提取error内部的值，所以有了这个函数，使用go1.26加入的errors.AsType将error转化为\*newerror.Error,即使类型不匹配也会将原始错误包装进\*newerror.Error
### AddTrace
\*newerror.Error的方法，用来向\*newerror.Error中的原始错误添加调用路径，常用方法为：
```txt
//函数的开头
 defer func(trace string){
    err=newrror.Translate(err).AddErrorTrace(trace)
 }("-")
```
这样可以快速定位到错误发生的地方
### WhetherInterrupt && WithContinueError
#### WhetherInterrupt
这个是项目中期时加入的，用于判断这个error有没有足够的“重量级”让请求终止。  
由于考虑到：缓存Redis出现问题，或者Kafka挂了，即使err!=nil,但是不影响正常流程，不应该因为err!=nil就return，所以通过这个进行判断
#### WithContinueError
用于在Make“轻量级”error时调用，这样生成的error不会触发WhetherInterrupt,可以让链路一直携带error，直到完成后“默默”打个日志
### MarshalError && UnMarshalError
这是项目后期调试期加入的，用于解决RPC框架只会传递err.Error()从而丢失其他信息的问题
#### MarshalError
考虑到序列化的开销，这个\*newerror.Error的方法选择以原始的方式：字符串拼接，进行序列化，生成一个原生error, error.Error()返回的是\*newerror.Error的JSON字符串。  
常用的用法：
```txt
\\RPC服务端的函数开头
defer func(){
    err=newerror.Translate(err).MarshalError()
}()
```
#### UnMarshalError
在RPC调用发接收到的error使用，将error中的JSON提取出来重新封装为\*newerror.Error, 即使反序列化失败，也会重新Make一个\*newerror.Error将原生的error封装在里面。  
常用的用法：
```text
resp,err:=serviceClient.DoSomething(ctx,req)
if newerror.WhetherInterrupt(newerror.UnMarshalError(err),&finalErr){
    return finalErr
}
```
反序列化选用的是bytedance的sonic包，通过运行时生成代码的方式，针对这种高频且对象确定的情况，开销远小于原生JSON库，使得与RPC相比，获取error的开销就微不足道了。  
同时方法内部会进行if err==nil{return nil}的方向，正常的业务逻辑不会触发反序列化，开销进一步减小。