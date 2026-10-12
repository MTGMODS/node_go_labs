# Lab 5 Backpressure Results

Source size: 100 MB. Slow consumer delay: 2 ms per chunk.

| Runtime | Mode | Size MB | Duration ms | Memory growth MB | Lines |
| --- | --- | ---: | ---: | ---: | ---: |
| Node.js | safe | 100 | 3911.32 | 9.72 | 1059168 |
| Node.js | unsafe | 100 | 3832.1 | 103.54 | 1059168 |
| Go | safe | 100 | 4567.41 | 3.26 | 1059168 |
| Go | unsafe | 100 | 5405.85 | 167.5 | 1059168 |
