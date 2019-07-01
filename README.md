# codegraph

Experiments with UAST graph.

## Installation

Get the latest [release](https://github.com/cayleygraph/cayley/releases) of Cayley, to be able to query the results. 

```bash
go install ./cmd/codegraph
```

## Running

You will need some UAST files in the YML format. You can get them from any of
[Babelfish drivers](https://github.com/search?utf8=%E2%9C%93&q=org%3Abblfsh+topic%3Ababelfish+topic%3Adriver&type=Repositories&ref=advsearch&l=&l=)
(see `./fixtures` folder).

Generate the graph data from UAST:
```bash
codegraph uast quads -o out.nq.gz ./fixtures/*.sem.uast
```

Import and run Cayley instance with the data:
```bash
cayley http -i out.nq.gz
```

Web interface should be available at http://127.0.0.1:64210.

## Queries

### All identifiers

**Gizmo** query language (Tinkerpop's Gremlin inspired):

```javascript
// Find an unknown node
g.V().
    // That has a specific type
    Has("<rdf:type>", "<uast:Identifier>").
    // Save the name of a node (apart from ID)
    Save("<uast:Name>", "name").
    // Limit and emit all the results
    Limit(100).All()
```

**GraphQL** inspired query language:

```graphql
{
  nodes(<rdf:type>: <uast:Identifier>, first: 100){
    id
    name: <uast:Name>
  }
}
```

### All imports

**Gizmo**:

```javascript
// Helpers to process different import path nodes.
// Can be embedded into the program later.
function toPath(m) {
  switch (m.type) {
  case "<uast:String>":
      return {path: g.V(m.id).Out("<uast:Value>").ToValue()}
  case "<uast:Identifier>":
      return {path: g.V(m.id).Out("<uast:Name>").ToValue()}
  case "<uast:Alias>":
      return toPath(g.V(m.id).Out("<uast:Node>").Save("<rdf:type>", "type").TagArray()[0]);
  case "<uast:QualifiedIdentifier>":
      var path = g.V(m.id).Out("<uast:Names>").Save("<rdf:type>", "type").TagArray();
      var s = ""
      for (i in path) {
        var n = toPath(path[i])
        if (i == 0) s = n.path
        else s += "/"+n.path
      }
      return {path: s}
  default:
      return m
  }
}

// Start from unknown node
g.V().
    // The node should be either Import or InlineImport
    Has("<rdf:type>", "<uast:Import>", "uast:InlineImport").
    // Traverse the Path field of the Import
    Out("<uast:Path>").
    // Save the type for the path node
    Save("<rdf:type>", "type").
    // Do not emit nodes directly, let JS function alter the data.
    // We will extract a single import path from any node type
    // that can be there.
    ForEach(function(m){
      g.Emit(toPath(m))
    })
```

**GraphQL** doesn't have a direct equivalent, since it cannot switch on the node type.
Instead we extract the path node, and optionally load path-related fields from the child node.

```graphql
{
  nodes(<rdf:type>: ["<uast:Import>", "<uast:InlineImport>"], first: 100){
    id
    path: <uast:Path> {
      type: <rdf:type>
      name: <uast:Name> @opt
      path: <uast:Value> @opt
      names: <uast:Names> @opt {
      name: <uast:Name>
      }
    }
  }
}
```

### All files that import "fmt"

**Gizmo**:

```javascript
var pkg = "fmt"

g.V().
    // Find the identifier that matches the package name
    Has("<rdf:type>", "<uast:Identifier>").
	Has("<uast:Name>", pkg).
    // Find node that points to this identifier with the Path predicate
	In("<uast:Path>").
    // Follow any relation backward recursively...
	FollowRecursive(g.V().In()).
    // until we hit a node with outgoing Root relation
    // (which goes from file to a UAST root).
    In("<uast:Root>").All()
```

Again, **GraphQL** doesn't have an equivalent.

### Import stats

**Gizmo**, _requires_ a helper from above:

```javascript
var imports = {}

g.V().
    Has("<rdf:type>", "<uast:Import>", "<uast:InlineImport>").
    Out("<uast:Path>").
    Save("<rdf:type>", "type").
    // Do not emit results, instead load the import path and collect counts.
    ForEach(function(m){
      var path = toPath(m).path;
      if (!imports[path]) imports[path] = 1;
      else imports[path]++;
    })

g.Emit(imports)
```