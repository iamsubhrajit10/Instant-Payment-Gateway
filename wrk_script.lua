wrk.method = 'POST'
wrk.headers['Content-Type'] = 'application/json'
counter = 0
request = function()
  counter = counter + 1
  local body = [[
{
    "Requests":[
        {
            "TransactionID": "]] .. counter .. [[",
            "PaymentID": "1",
            "Type": "resolve"
        },
        {
            "TransactionID": "]] .. counter .. [[",
            "PaymentID": "2",
            "Type": "resolve"
        }
    ]
}
]]
  return wrk.format(nil, nil, nil, body)
end
