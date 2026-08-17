resource "aws_s3_bucket" "test" {
  bucket = "my-test-bucket"
  policy = jsonencode({
    Statement = [{
      Condition = { StringEquals = { "s3:tls1.0" = "true" } }
    }]
  })
}
