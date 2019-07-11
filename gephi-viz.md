# Visualizing the graph

**Prerequisites:**

- codegraph
- Gephi 0.9+
- Cayley 0.7+

## Generating the graph

First, you need to generate a quad file for the graph. For Git repo, you can run:

```bash
$ codegraph git quads -o out.nq.gz ./some-repo
```

This will generate a `out.nq.gz` file containing quads for a given repository as well as Gephi-related metadata.

Not that the file extension _is important_ and will be used by Cayley to select the plugin to load the data.

## Running Cayley server

The simples way to load the quad file is to start a server with in-memory backend:

```bash
$ cayley http -i out.nq.gz
```

After a few seconds, wou should be able to open the [http://localhost:64320](http://localhost:64320) and see the Cayley UI.

Although you can query the graph this way, we won't use it for visualization. Leave the server running and start Gephi.

## Gephi

You can get Gephi from the [official page](https://gephi.org/).

After installing, run it and go to the "Tools" -> "Plugins" -> "Available Plugins" and install "Graph Streaming".
We will use this plugin to communicate with Cayley instance. You may need to restart Gephi in order to load the plugin.

Next, create a new project/workspace and click the "Streaming tab" in the middle of the left panel (near the "Layout" tab).
You should see the "Client" and the "Master" items in the list.

Right click the "Client" item, click "Connect ...", check that the stream type is set to JSON and enter the following URL as an address:

`http://127.0.0.1:64210/gephi/gs?mode=nodes&limit=-1`

After clicking "OK", Gephi will start loading the data from Cayley instance. This may take a bit more time, make sure that
the icon near the "Client" item becomes red before doing anything with the graph.

## Cleaning up the data

For now, you need to manually wipe a few super-nodes and auxiliary relations to make the graph more user-friendly.

Go to the "Data lab" tab at the top toolbar, select "Nodes" tab. You will see the first node that matches the repository URL.
Click on the row and delete it.

Switch to the "Edges" tab, enter `git:file` in the filter box at the right top, select `pred` as a field.
Then, select all the edges that matched the filter (`Ctrl+A`) and delete them.
You may want to do the same for `git:commiter`, if you only want an author of commits to appear.

Now you are ready to run the layout.

## Layout

Switch back to the "Overview" tab at the toolbar, the to "Layout" tab on the left panel (near the "Client"), select "Force Atlas 2" as the layout engine.

If your graph is very large, set the "Approximate repulsion" flag.

Run the layout for a few seconds. You may want to adjust the "Scale" to make it more sparse or more condensed.
"LinLog mode" may also help with this task.

After you are happy with parameters, stop the layout engine.

## Visuals

To tune visuals, switch to the "Appearance" tab at the top-left, and to the "Nodes" sub-tab.
Click "Partition" and select `<rdf:type>` as an attribute. Apply the changes.

You can do the same for the edges by selecting `pred` attribute in the corresponding tab.

It also makes sense to rank node size and text size by in-degree (see icons on the right of the tab names).