resource "aws_vpc" "eks-vpc" {
  cidr_block       = "10.0.0.0/16"
  instance_tenancy = "default"
}

resource "aws_subnet" "eks-subnet" {
  vpc_id     = aws_vpc.eks-vpc.id
  cidr_block = "10.0.1.0/24"
}
