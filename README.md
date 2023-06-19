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
