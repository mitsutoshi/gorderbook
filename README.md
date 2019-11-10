gorderbook
==========

gorderbook is tool to monitor bitFlyer's board(order book).

gorderbook monitor the bitflyer's board. You can display grouped price and size.

![A](./image.png)

## Usage

    Usage of ./gorderbook:
      -group int
            grouping price on board. (default 1)


### e.g.

Don't grouping.

```
gorderbook
```

Group every 50 yen.

```
gorderbook -group 50
```

![A](./image-group50.png)
