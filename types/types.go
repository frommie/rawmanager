package types

import "encoding/xml"

type XmpMeta struct {
    XMLName xml.Name `xml:"xmpmeta"`
    RDF     struct {
        Description struct {
            Rating    string `xml:"http://ns.adobe.com/xap/1.0/ Rating"`
            MSRating  string `xml:"http://ns.microsoft.com/photo/1.0/ Rating"`
        } `xml:"Description"`
    } `xml:"RDF"`
}