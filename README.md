tinynfs
==============

`tinynfs`  A small file & image storage system

## Getting Started

### File Storage

### Image Storage

Supported type: **gif**, **jpeg**, **png**

#### Upload One Image File

##### Request

``` bash
curl -X POST \
  http://127.0.0.1:7120/upload \
  -F imagedata=@/Users/vietor/Pictures/demo.jpg
```

##### Response

``` json
{
    "code": 0,
    "data": {
        "size": 60133,
        "width": 312,
        "height": 304
        "image_url": "/image1/c2320d8876dfcbbf715f5b8f40e3",
    }
}
```

#### Upload Multiple Image File

##### Request

``` bash
curl -X POST \
  http://127.0.0.1:7120/uploads \
  -F imagedata1=@/Users/vietor/Pictures/demo1.jpg \
  -F imagedata2=@/Users/vietor/Pictures/demo2.jpg \
  -F imagedata3=@/Users/vietor/jmeter.log
```

##### Response

``` json
{
    "code": 0,
    "data": {
        "imagedata1": {
            "height": 1386,
            "image_url": "/image1/f9efdbbffa65ba17f1a45b8ea9a8",
            "size": 409375,
            "width": 972
        },
        "imagedata2": {
            "height": 487,
            "image_url": "/image1/4bf143e55eae3d3d51955b8ea9a8",
            "size": 43735,
            "width": 435
        },
        "imagedata3": {
            "error": "unsupported media type"
        }
    }
}
```

#### Get Image

##### Origin Image

The image url was reponsed by `/upload` or `/uploads`

```
http://127.0.0.1:7120/image1/c2320d8876dfcbbf715f5b8f40e3
```

##### Thumbnail Image

The origin image url add the "_" and thumbnail size stuffix

> the gif transform to png

```
http://127.0.0.1:7120/image1/c2320d8876dfcbbf715f5b8f40e3_192x192
```

The acceptable thumbnail size was defined in configuration file

```
# network.image.thumbnail.sizes=240x240,192x192
```

