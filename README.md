# GoNetReader
## NetReader for net.Conn

### Example:
#### Read:
```
func ConnectionRouter(Connection net.Conn) {
  reader := GoNetReader.NewNetReader()
  for {
    Pack, err := reader.NetRead(Connection)
      if err != nil {
	if err == io.EOF {
          Connection.Close() // EOF
        }
        continue 
      }
    //  ...
  }
  // ...
}
```  
#### Write:
``` 
func Sender(Connection net.Conn) { 
  // ...
  _, connecterr := Connection.Write(GoNetReader.GetPackage(pack))
  // ...
}

```  
### Scheme:
![](sheme.jpg)
