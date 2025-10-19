# Distributed Display Protocol (DDP) in Go

This package allows you to write pixel data to a LED strip over [DDP](http://www.3waylabs.com/ddp/)
Currently implements sending but haven't gotten to implementing response parsing, works for most use cases though.

You can use this to stream pixel data to [WLED](https://github.com/Aircoookie/WLED) or any other DDP capable reciever.

## Simple Example

```go
// Create a new DDP client
client := ddp.NewDDPController()

// Connect to DDP server over UDP
client.ConnectUDP("10.0.1.9:4048")

//Write one pixel
written, err := client.Write([]byte{128, 36, 12})
fmt.Println(written, err)
```

## Contributing

m8 just open a PR with some gucchimucchi code and I'll review it.

![KADSBUGGEL](https://raw.githubusercontent.com/coral/fluidsynth2/master/kadsbuggel.png)

## AI DISCLAIMER

I initially wrote this by hand but used AI SLOP to SLOP IT UP for testing etc. Everything pre 2025 is A R T I S I N A L but now we're fully in slop city. **no warranties express or implied** etc etc. TBH mostly tests were slopped.