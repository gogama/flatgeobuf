# flatgeobuf
Flatgeobuf binary geospatial encoding in Native Go



Planned package structure:
    1. In root
            1) I will auto-(re)generate Flatbuffers from fbs source,
               because flatbuffers git repo Go code is not a module
               and is weirdly nested.
            2) I'll also have the hand-written Reader and Writer
               structs, one per file.
            3) I don't anticipate needing to handle any endian-ness
               issues as I expect the Flatbuffers library will handle
               this.
    2. In packedrtree, there will be appropriate little-endian conversion
       go code which is only needed for serde for the index section.
