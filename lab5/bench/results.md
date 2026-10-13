# Lab 5 Backpressure Results

Source size: 100 MB. Slow consumer delay: 2 ms per chunk.

| Runtime | Mode | Size MB | Duration ms | Memory growth MB | Lines |
| --- | --- | ---: | ---: | ---: | ---: |
| Node.js | safe | 100 | 3646.75 | 8.25 | 1059168 |
| Node.js | unsafe | 100 | 4459.37 | 74.81 | 1059168 |
| Go | safe | 100 | 6039.16 | 3.19 | 1059168 |
| Go | unsafe | 100 | 7940.30 | 191.84 | 1059168 |
