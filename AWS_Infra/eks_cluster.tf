resource "aws_eks_cluster" "jc-eks" {
  name     = "jc-cluster"
  role_arn = aws_iam_role.eks-iam-role.arn

  vpc_config {
    subnet_ids = [aws_subnet.public_subnets[0].id,aws_subnet.public_subnets[1].id, aws_subnet.public_subnets[2].id]
  }

  depends_on = [
    aws_iam_role.eks-iam-role,
  ]
}

resource "aws_eks_node_group" "worker-node-group" {
  cluster_name    = aws_eks_cluster.jc-eks.name
  node_group_name = "jc-workernodes"
  node_role_arn   = aws_iam_role.workernodes.arn
  subnet_ids      = [aws_subnet.public_subnets[0].id,aws_subnet.public_subnets[1].id, aws_subnet.public_subnets[2].id]
  instance_types  = ["t3a.small"]

  scaling_config {
    desired_size = 1
    max_size     = 1
    min_size     = 1
  }

  depends_on = [
    aws_iam_role_policy_attachment.AmazonEKSWorkerNodePolicy,
    aws_iam_role_policy_attachment.AmazonEKS_CNI_Policy,
    #aws_iam_role_policy_attachment.AmazonEC2ContainerRegistryReadOnly,
  ]
}
