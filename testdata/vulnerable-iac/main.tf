resource "aws_s3_bucket" "my_data" {
  bucket = "my-sensitive-data"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = "s3:GetObject"
        Principal = "*"
        Resource = "arn:aws:s3:::my-sensitive-data/*"
        Condition = {
          StringEquals = {
            "s3:tls1.0" = "true"
          }
        }
      }
    ]
  })
}
