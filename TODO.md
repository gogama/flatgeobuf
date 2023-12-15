NEXT STEPS:
    1. Finish unit testing package flatgeobuf.
    2. Since the go:embed thing didn't work, my next strategem is to
       use a go:generate type script to physically embed the bytes
       in `example_test.go`.
    3. If I'm committing to Go 1.20 due to unsafe.String, then:
        (a) replace all interface{} with any.
        (b) consider using a generics-based heap which is faster? Or
            maybe do that another day.
    4. Cut v0.9.5-beta.
    5. Add GitHub action to build/test on commit? (If free.)
    6. Update copyrights to 2024. 
    7. Clear out any remaining TODO, for e.g. example_test.go. 
    8. Cut v1.0.0.

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
   consumers don't need orb. An alternative is just to create another
   repository, flatgeobuf-orb, for this extended functionality.
