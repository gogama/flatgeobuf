# flatgeobuf
FlatGeobuf binary geospatial encoding in Native Go

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
    1. If I'm committing to Go 1.20 due to unsafe.String, then:
        - replace all interface{} with any.
        - Consider using a generics-based heap which is faster? Or maybe do that another day.
    2. Clear up all CODE and DOCUMENTATION references to Ref.Offset and
       validate that it works. I think the code and docs are ambiguous
       or assumey about whether offset is relative to data section start
       or relative to file start.
    3. I would restructure things to move most of the root files to a
       subdirectory, so:
        flatgeobuf/  ---> Main code, well-documented
           flat/ ---> Generated code, only package.go has docs
        packedrtree/ ---> Packed R-Tree
        testdata/ ---> Test data
        tmp/flatgeobuf/ ---> Temporary directory where upstream gets checked out by scripts.
