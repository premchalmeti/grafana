UPDATE {{ .Ident "secure_value" }} SET 
 "salt"=?, "value"=?, 
 "keeper"=?, "addr"=?,
 "updated"=?, "updated_by"=?,
 "annotations"=?, "labels"=?, 
 "apis"=?
WHERE "uid"=?;