NEXT STEPS:
    1. Fix stateful issues and finalize.
    2. Cut v0.9.0-alpha.
    3. Request feedback.
    4. Finish unit testing package flatgeobuf.
    5. If possible, fix openFile to perhaps embed the test data files
       if that will make the examples run on pkg.go.dev?
    6. Cut v0.9.5-beta.
    7. Cut v1.0.0.

Future directions:
1. Another interesting interaction system would be an Appender which
can be used to append features to a non-indexed existing FlatGeobuf
file. This would have to be implemented on top of an io.ReadWriteSeeker,
where you would read the magic and header, then jump to the end of the
file and append while updating the feature count in the header.
This would address the "append without index" use case suggested on the FlatGeobuf
docs site.
2. Add an orb compatibility package for feature conversion? The orb
   dependency would be ring-fenced to this sub-package so that other
   consumers don't need orb.

TODO:
1. If I'm committing to Go 1.20 due to unsafe.String, then:
- replace all interface{} with any.
- Consider using a generics-based heap which is faster? Or maybe do that another day.
2. Clear up all CODE and DOCUMENTATION references to Ref.Offset and
validate that it works. I think the code and docs are ambiguous
or assumey about whether offset is relative to data section start
or relative to file start.
