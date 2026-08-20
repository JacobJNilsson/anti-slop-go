package a

import _ "reflect" // want `this package imports reflect`

// A blank import of reflect uses no object of the package. A file that
// is no test file reports it anyway, because the import is the
// decision and the file needs no reflection at all.
