# flatgeobuf
FlatGeobuf binary geospatial encoding in Native Go

Mandatory now:
   1. A packer/unpacker for Feature properties is needed. The way
      properties work is overly cryptic.
   2. The following types should have a String() method:
        (A) Header
        (B) Feature
      This will clean up the code and output of the example tests... 

Work to be done:
done    2) Create PropReader, PropWriter in prop_reader.go, prop_writer.go
            Reader: always takes a schema.
                        Has methods like ReadJSON(col int), ReadInt(col int).
                        Also has ReadRem() -> []PropValue where prop value is struct { ColIndex int, Col *Column, Value interface{} }
            Writer: either takes or produces a schema.
                        Always takes a schema ([]Columns) and all writes have to match it.
                        Even if a feature has ad hoc columns, schema sold separately.
                        Has methods like WriteJSON(col int, json []byte).
    3) Add string.go to generate String() methods for Feature and Header.

Future directions:
    1. Another interesting interaction system would be an Appender which
       can be used to append features to a non-indexed existing FlatGeobuf
       file. This would have to be implemented on top of an io.ReadWriteSeeker,
       where you would read the magic and header, then jump to the end of the
       file and append while updating the feature count in the header.
       This would address the "append without index" use case suggested on the FlatGeobuf
       docs site.

TODO in README:
   1. Ensure credit is given for testdata/flatgeobuf/*.
   2. Ensure credit is given for FlatGeobuf format and schema FBS.
   3. Generally ensure attribution of all the things.

TODO:
    1. If I'm committing to Go 1.20 due to unsafe.String, then replace
       all interface{} with any.
