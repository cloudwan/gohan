Using NPM modules in extensions
-------------------------------

In this example, we show how we can
use modules installed via npm in js extensions

We have installed npm module underscore in 
node_modules at working directory.

``` js
        var underscore = require('underscore');
        console.log(underscore.min([3,2,1]));
```

We use require function to load module. Then we can 
use it.